// Package export reverse-engineers a live Kubernetes cluster into a clean,
// structured IaC git repository. Output is deployable immediately after
// filling in secret placeholders.
package export

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/JunaDev/cortixlabs/internal/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

// ExportOptions controls the export behaviour.
type ExportOptions struct {
	Namespace        string // "" = all non-system namespaces
	OutputDir        string // default: ./cortix-export
	Format           string // kustomize | helm | external-secrets
	AddBestPractices bool
	DryRun           bool
}

// ExportedResource is a single resource ready to write as a YAML file.
type ExportedResource struct {
	Name     string
	Filename string
	Content  []byte // clean YAML, cluster noise stripped
}

// NamespaceExport holds all exported resources for one namespace.
type NamespaceExport struct {
	Name            string
	Deployments     []ExportedResource
	StatefulSets    []ExportedResource
	Services        []ExportedResource
	Ingresses       []ExportedResource
	ConfigMaps      []ExportedResource
	Secrets         []ExportedResource
	ServiceAccounts []ExportedResource
	HPAs            []ExportedResource
	HelmReleases    []HelmRelease
	Warnings        []string
}

// ClusterExport holds cluster-wide resources.
type ClusterExport struct {
	Namespaces          []ExportedResource
	StorageClasses      []ExportedResource
	ClusterRoles        []ExportedResource
	ClusterRoleBindings []ExportedResource
	Warnings            []string
}

// ExportResult is the full in-memory export, ready to write to disk.
type ExportResult struct {
	ClusterName  string
	Cluster      ClusterExport
	Namespaces   []NamespaceExport
	Warnings     []string
	SecretCount  int // total secrets sanitized
	HelmCount    int // total Helm releases detected
}

// Exporter orchestrates the full reverse-YAML export.
type Exporter struct {
	client *k8s.Client
}

// New creates an Exporter backed by client.
func New(client *k8s.Client) *Exporter {
	return &Exporter{client: client}
}

// Run fetches all cluster resources, strips noise, sanitizes secrets,
// applies best practices if requested, then writes the export structure.
func (e *Exporter) Run(ctx context.Context, opts ExportOptions) (*ExportResult, error) {
	if opts.OutputDir == "" {
		opts.OutputDir = "./cortix-export"
	}
	slog.Info("starting export",
		"cluster", e.client.ClusterName,
		"namespace", opts.Namespace,
		"outputDir", opts.OutputDir,
		"dryRun", opts.DryRun,
	)

	result := &ExportResult{ClusterName: e.client.ClusterName}

	// Cluster-wide resources
	cluster, err := e.fetchClusterResources(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("export: cluster resources: %w", err)
	}
	result.Cluster = *cluster

	// Resolve namespaces to export
	namespaceNames, err := e.resolveNamespaces(ctx, opts.Namespace)
	if err != nil {
		return nil, fmt.Errorf("export: resolve namespaces: %w", err)
	}

	for _, ns := range namespaceNames {
		nsExport, err := e.fetchNamespaceResources(ctx, ns, opts)
		if err != nil {
			slog.Warn("skipping namespace", "namespace", ns, "err", err)
			result.Warnings = append(result.Warnings, fmt.Sprintf("namespace %s: %v", ns, err))
			continue
		}
		result.SecretCount += len(nsExport.Secrets)
		result.HelmCount += len(nsExport.HelmReleases)
		result.Namespaces = append(result.Namespaces, *nsExport)
	}

	// Write to disk unless dry-run
	if !opts.DryRun {
		w := NewWriter(opts.OutputDir)
		if err := w.Write(result); err != nil {
			return nil, fmt.Errorf("export: write: %w", err)
		}
		slog.Info("export complete", "outputDir", opts.OutputDir,
			"namespaces", len(result.Namespaces),
			"secrets", result.SecretCount,
			"helmReleases", result.HelmCount,
		)
	}

	return result, nil
}

// resolveNamespaces returns the namespaces to export.
// If namespace is set, returns just that one. Otherwise returns all non-system namespaces.
func (e *Exporter) resolveNamespaces(ctx context.Context, namespace string) ([]string, error) {
	if namespace != "" {
		return []string{namespace}, nil
	}

	nsList, err := e.client.Typed.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}

	systemNS := map[string]bool{
		"kube-system": true, "kube-public": true, "kube-node-lease": true,
	}

	var names []string
	for _, ns := range nsList.Items {
		if !systemNS[ns.Name] {
			names = append(names, ns.Name)
		}
	}
	return names, nil
}

// fetchClusterResources fetches cluster-wide resources (RBAC, StorageClasses, Namespaces).
func (e *Exporter) fetchClusterResources(ctx context.Context, opts ExportOptions) (*ClusterExport, error) {
	ce := &ClusterExport{}

	// Namespaces
	nsItems, err := e.listDynamic(ctx, gvrNamespace, "")
	if err != nil {
		ce.Warnings = append(ce.Warnings, "could not list namespaces: "+err.Error())
	}
	for _, obj := range nsItems {
		obj := obj
		res, err := toExportedResource(&obj, opts)
		if err != nil {
			continue
		}
		ce.Namespaces = append(ce.Namespaces, *res)
	}

	// StorageClasses
	scItems, err := e.listDynamic(ctx, gvrStorageClass, "")
	if err != nil {
		ce.Warnings = append(ce.Warnings, "could not list storage classes: "+err.Error())
	}
	for _, obj := range scItems {
		obj := obj
		res, err := toExportedResource(&obj, opts)
		if err != nil {
			continue
		}
		ce.StorageClasses = append(ce.StorageClasses, *res)
	}

	// ClusterRoles (skip system: prefixed)
	crItems, err := e.listDynamic(ctx, gvrClusterRole, "")
	if err != nil {
		ce.Warnings = append(ce.Warnings, "could not list cluster roles: "+err.Error())
	}
	for _, obj := range crItems {
		obj := obj
		if strings.HasPrefix(obj.GetName(), "system:") {
			continue
		}
		res, err := toExportedResource(&obj, opts)
		if err != nil {
			continue
		}
		ce.ClusterRoles = append(ce.ClusterRoles, *res)
	}

	// ClusterRoleBindings (skip system: prefixed)
	crbItems, err := e.listDynamic(ctx, gvrClusterRoleBinding, "")
	if err != nil {
		ce.Warnings = append(ce.Warnings, "could not list cluster role bindings: "+err.Error())
	}
	for _, obj := range crbItems {
		obj := obj
		if strings.HasPrefix(obj.GetName(), "system:") {
			continue
		}
		res, err := toExportedResource(&obj, opts)
		if err != nil {
			continue
		}
		ce.ClusterRoleBindings = append(ce.ClusterRoleBindings, *res)
	}

	return ce, nil
}

// fetchNamespaceResources fetches all relevant resources for one namespace.
func (e *Exporter) fetchNamespaceResources(ctx context.Context, ns string, opts ExportOptions) (*NamespaceExport, error) {
	ne := &NamespaceExport{Name: ns}
	helmSeen := map[string]bool{}

	type listJob struct {
		gvr    schema.GroupVersionResource
		target *[]ExportedResource
		skip   func(*unstructured.Unstructured) bool
	}

	jobs := []listJob{
		{gvrDeployment, &ne.Deployments, nil},
		{gvrStatefulSet, &ne.StatefulSets, nil},
		{gvrService, &ne.Services, isSystemService},
		{gvrIngress, &ne.Ingresses, nil},
		{gvrHPA, &ne.HPAs, nil},
		{gvrServiceAccount, &ne.ServiceAccounts, isDefaultServiceAccount},
	}

	for _, job := range jobs {
		items, err := e.listDynamic(ctx, job.gvr, ns)
		if err != nil {
			ne.Warnings = append(ne.Warnings, fmt.Sprintf("list %s: %v", job.gvr.Resource, err))
			continue
		}
		for _, obj := range items {
			obj := obj

			// Route Helm-managed resources to HelmReleases
			if IsHelmManaged(&obj) {
				rel := ExtractHelmRelease(&obj)
				if !helmSeen[rel.Name] {
					helmSeen[rel.Name] = true
					ne.HelmReleases = append(ne.HelmReleases, rel)
				}
				continue
			}

			if job.skip != nil && job.skip(&obj) {
				continue
			}

			res, err := toExportedResource(&obj, opts)
			if err != nil {
				ne.Warnings = append(ne.Warnings, fmt.Sprintf("marshal %s/%s: %v", job.gvr.Resource, obj.GetName(), err))
				continue
			}
			*job.target = append(*job.target, *res)
		}
	}

	// ConfigMaps — skip system ones
	cmItems, _ := e.listDynamic(ctx, gvrConfigMap, ns)
	for _, obj := range cmItems {
		obj := obj
		if isSystemConfigMap(&obj) {
			continue
		}
		if IsHelmManaged(&obj) {
			continue
		}
		res, err := toExportedResource(&obj, opts)
		if err != nil {
			continue
		}
		ne.ConfigMaps = append(ne.ConfigMaps, *res)
	}

	// Secrets — always sanitize, never write real values
	secretItems, _ := e.listDynamic(ctx, gvrSecret, ns)
	for _, obj := range secretItems {
		obj := obj
		if isSystemSecret(&obj) {
			continue
		}
		sanitized := SanitizeSecret(&obj)
		res, err := toExportedResource(sanitized, opts)
		if err != nil {
			continue
		}
		ne.Secrets = append(ne.Secrets, *res)
	}

	return ne, nil
}

// listDynamic lists all resources of a given GVR in a namespace ("" = cluster-scoped).
func (e *Exporter) listDynamic(ctx context.Context, gvr schema.GroupVersionResource, namespace string) ([]unstructured.Unstructured, error) {
	var list *unstructured.UnstructuredList
	var err error

	if namespace == "" {
		list, err = e.client.Dynamic.Resource(gvr).List(ctx, metav1.ListOptions{})
	} else {
		list, err = e.client.Dynamic.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// toExportedResource strips, optionally enriches, and marshals one resource.
func toExportedResource(obj *unstructured.Unstructured, opts ExportOptions) (*ExportedResource, error) {
	stripped := Strip(obj)

	if opts.AddBestPractices {
		stripped = ApplyBestPractices(stripped)
	}

	content, err := marshalYAML(stripped)
	if err != nil {
		return nil, err
	}

	return &ExportedResource{
		Name:     obj.GetName(),
		Filename: obj.GetName() + ".yaml",
		Content:  content,
	}, nil
}

// marshalYAML converts an unstructured object to clean YAML bytes.
func marshalYAML(obj *unstructured.Unstructured) ([]byte, error) {
	j, err := obj.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}
	y, err := yaml.JSONToYAML(j)
	if err != nil {
		return nil, fmt.Errorf("convert yaml: %w", err)
	}
	return y, nil
}

// --- GVR registry ---

var (
	gvrDeployment         = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	gvrStatefulSet        = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}
	gvrService            = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	gvrIngress            = schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}
	gvrConfigMap          = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	gvrSecret             = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
	gvrServiceAccount     = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}
	gvrHPA                = schema.GroupVersionResource{Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"}
	gvrNamespace          = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}
	gvrClusterRole        = schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"}
	gvrClusterRoleBinding = schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings"}
	gvrStorageClass       = schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses"}
)

// --- Skip filters ---

func isSystemService(obj *unstructured.Unstructured) bool {
	return obj.GetName() == "kubernetes"
}

func isDefaultServiceAccount(obj *unstructured.Unstructured) bool {
	return obj.GetName() == "default"
}

func isSystemConfigMap(obj *unstructured.Unstructured) bool {
	name := obj.GetName()
	return name == "kube-root-ca.crt" || strings.HasPrefix(name, "kube-")
}

func isSystemSecret(obj *unstructured.Unstructured) bool {
	t, _, _ := unstructured.NestedString(obj.Object, "type")
	return t == "kubernetes.io/service-account-token"
}
