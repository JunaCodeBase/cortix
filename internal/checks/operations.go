package checks

import (
	"context"
	"fmt"

	"github.com/JunaDev/cortixlabs/internal/k8s"
	"github.com/JunaDev/cortixlabs/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1 "k8s.io/api/apps/v1"
)

// OperationsChecks returns all operations checks in priority order.
func OperationsChecks() []Check {
	return []Check{
		NoRollingUpdateStrategy{},
		DefaultNamespaceInProduction{},
		CertManagerCheck{},
		IngressWithTLS{},
		StorageClassDefined{},
		HPAPoliciesConfigured{},
	}
}

// --- ops-001: No rolling update strategy ---

type NoRollingUpdateStrategy struct{}

func (NoRollingUpdateStrategy) ID() string              { return "ops-001" }
func (NoRollingUpdateStrategy) Name() string             { return "No rolling update strategy" }
func (NoRollingUpdateStrategy) Category() types.Category { return types.CategoryOperations }

func (chk NoRollingUpdateStrategy) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	deps, err := client.Typed.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: list deployments: %w", chk.ID(), err)
	}

	var results []types.CheckResult
	for _, d := range deps.Items {
		if d.Spec.Strategy.Type != appsv1.RollingUpdateDeploymentStrategyType {
			results = append(results, types.CheckResult{
				ID:          chk.ID(),
				Category:    chk.Category(),
				Name:        chk.Name(),
				Severity:    types.SeverityWarning,
				Namespace:   d.Namespace,
				Resource:    fmt.Sprintf("deployment/%s", d.Name),
				Detail:      fmt.Sprintf("strategy: %s — pods are replaced all at once, causing downtime", d.Spec.Strategy.Type),
				Remediation: "Set spec.strategy.type: RollingUpdate with maxSurge and maxUnavailable values.",
			})
		}
	}
	if len(results) == 0 {
		return pass(chk.ID(), chk.Name(), chk.Category()), nil
	}
	return results, nil
}

// --- ops-002: Production workloads in default namespace ---

type DefaultNamespaceInProduction struct{}

func (DefaultNamespaceInProduction) ID() string              { return "ops-002" }
func (DefaultNamespaceInProduction) Name() string             { return "Production workloads in default namespace" }
func (DefaultNamespaceInProduction) Category() types.Category { return types.CategoryOperations }

func (chk DefaultNamespaceInProduction) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	pods, err := client.Typed.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: list pods in default namespace: %w", chk.ID(), err)
	}

	if len(pods.Items) > 0 {
		return []types.CheckResult{{
			ID:          chk.ID(),
			Category:    chk.Category(),
			Name:        chk.Name(),
			Severity:    types.SeverityWarning,
			Namespace:   "default",
			Resource:    fmt.Sprintf("%d pods running in default namespace", len(pods.Items)),
			Detail:      "the default namespace has no RBAC, no quotas, and no network isolation by default",
			Remediation: "Move all production workloads to dedicated namespaces with RBAC, quotas, and NetworkPolicies.",
		}}, nil
	}
	return pass(chk.ID(), chk.Name(), chk.Category()), nil
}

// --- ops-003: cert-manager ---

type CertManagerCheck struct{}

func (CertManagerCheck) ID() string              { return "ops-003" }
func (CertManagerCheck) Name() string             { return "cert-manager — TLS automation" }
func (CertManagerCheck) Category() types.Category { return types.CategoryOperations }

func (chk CertManagerCheck) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	pods, err := client.Typed.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=cert-manager",
		Limit:         1,
	})
	if err != nil {
		return nil, err
	}

	if len(pods.Items) == 0 {
		return []types.CheckResult{{
			ID:          chk.ID(),
			Category:    chk.Category(),
			Name:        chk.Name(),
			Severity:    types.SeverityWarning,
			Passed:      false,
			Detail:      "No TLS certificate automation. Certificates must be renewed manually — risk of outage on expiry.",
			Remediation: "helm install cert-manager jetstack/cert-manager -n cert-manager --create-namespace --set installCRDs=true",
		}}, nil
	}

	return pass(chk.ID(), chk.Name(), chk.Category()), nil
}

// --- ops-004: Ingress with TLS ---

type IngressWithTLS struct{}

func (IngressWithTLS) ID() string              { return "ops-004" }
func (IngressWithTLS) Name() string             { return "Ingress without TLS" }
func (IngressWithTLS) Category() types.Category { return types.CategoryOperations }

func (chk IngressWithTLS) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	ingresses, err := client.Typed.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: list ingresses: %w", chk.ID(), err)
	}

	var results []types.CheckResult
	for _, ing := range ingresses.Items {
		if len(ing.Spec.TLS) == 0 {
			results = append(results, types.CheckResult{
				ID:          chk.ID(),
				Category:    chk.Category(),
				Name:        chk.Name(),
				Severity:    types.SeverityWarning,
				Namespace:   ing.Namespace,
				Resource:    fmt.Sprintf("ingress/%s", ing.Name),
				Detail:      "ingress has no TLS configuration — traffic is served over HTTP",
				Remediation: "Add a spec.tls block with a cert-manager Certificate or a manually managed TLS secret.",
			})
		}
	}
	if len(results) == 0 {
		return pass(chk.ID(), chk.Name(), chk.Category()), nil
	}
	return results, nil
}

// --- ops-005: StorageClass defined ---

type StorageClassDefined struct{}

func (StorageClassDefined) ID() string              { return "ops-005" }
func (StorageClassDefined) Name() string             { return "No default StorageClass defined" }
func (StorageClassDefined) Category() types.Category { return types.CategoryOperations }

func (chk StorageClassDefined) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	classes, err := client.Typed.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: list storage classes: %w", chk.ID(), err)
	}

	for _, sc := range classes.Items {
		if sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
			return pass(chk.ID(), chk.Name(), chk.Category()), nil
		}
	}

	return []types.CheckResult{{
		ID:          chk.ID(),
		Category:    chk.Category(),
		Name:        chk.Name(),
		Severity:    types.SeverityImprovement,
		Detail:      "no default StorageClass — PVCs without an explicit storageClassName will remain unbound",
		Remediation: "Annotate a StorageClass with storageclass.kubernetes.io/is-default-class: \"true\".",
	}}, nil
}

// --- ops-006: HPA policies configured ---

type HPAPoliciesConfigured struct{}

func (HPAPoliciesConfigured) ID() string              { return "ops-006" }
func (HPAPoliciesConfigured) Name() string             { return "HPA policies configured" }
func (HPAPoliciesConfigured) Category() types.Category { return types.CategoryOperations }

func (chk HPAPoliciesConfigured) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	hpas, err := client.Typed.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: list HPAs: %w", chk.ID(), err)
	}

	if len(hpas.Items) == 0 {
		return []types.CheckResult{{
			ID:          chk.ID(),
			Category:    chk.Category(),
			Name:        chk.Name(),
			Severity:    types.SeverityImprovement,
			Detail:      "no HorizontalPodAutoscalers found — workloads cannot scale under load automatically",
			Remediation: "Add HPA resources with CPU/memory targets for stateless workloads.",
		}}, nil
	}

	return pass(chk.ID(), chk.Name(), chk.Category()), nil
}
