package checks

import (
	"context"
	"fmt"

	"github.com/JunaCodeBase/cortix/internal/k8s"
	"github.com/JunaCodeBase/cortix/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecurityChecks returns all security checks in priority order.
func SecurityChecks() []Check {
	return []Check{
		PodsRunningAsRoot{},
		PrivilegedContainers{},
		SecretsAsEnvVars{},
		MissingNetworkPolicies{},
		PublicImageRegistries{},
		RBACWildcardPermissions{},
		HostNetworkHostPID{},
	}
}

// --- sec-001: Pods running as root ---

type PodsRunningAsRoot struct{}

func (PodsRunningAsRoot) ID() string              { return "sec-001" }
func (PodsRunningAsRoot) Name() string             { return "Pods running as root" }
func (PodsRunningAsRoot) Category() types.Category { return types.CategorySecurity }

func (chk PodsRunningAsRoot) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	pods, err := client.Typed.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: list pods: %w", chk.ID(), err)
	}

	var results []types.CheckResult
	for _, pod := range pods.Items {
		for _, ctr := range pod.Spec.Containers {
			if ctr.SecurityContext != nil &&
				ctr.SecurityContext.RunAsNonRoot != nil &&
				!*ctr.SecurityContext.RunAsNonRoot {
				results = append(results, types.CheckResult{
					ID:          chk.ID(),
					Category:    chk.Category(),
					Name:        chk.Name(),
					Severity:    types.SeverityCritical,
					Namespace:   pod.Namespace,
					Resource:    fmt.Sprintf("pod/%s container/%s", pod.Name, ctr.Name),
					Detail:      "runAsNonRoot is explicitly false",
					Remediation: "Set securityContext.runAsNonRoot: true and securityContext.runAsUser to a non-zero UID.",
				})
			}
		}
	}
	if len(results) == 0 {
		return pass(chk.ID(), chk.Name(), chk.Category()), nil
	}
	return results, nil
}

// --- sec-002: Privileged containers ---

type PrivilegedContainers struct{}

func (PrivilegedContainers) ID() string              { return "sec-002" }
func (PrivilegedContainers) Name() string             { return "Privileged containers" }
func (PrivilegedContainers) Category() types.Category { return types.CategorySecurity }

func (chk PrivilegedContainers) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	pods, err := client.Typed.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: list pods: %w", chk.ID(), err)
	}

	var results []types.CheckResult
	for _, pod := range pods.Items {
		for _, ctr := range pod.Spec.Containers {
			if ctr.SecurityContext != nil &&
				ctr.SecurityContext.Privileged != nil &&
				*ctr.SecurityContext.Privileged {
				results = append(results, types.CheckResult{
					ID:          chk.ID(),
					Category:    chk.Category(),
					Name:        chk.Name(),
					Severity:    types.SeverityCritical,
					Namespace:   pod.Namespace,
					Resource:    fmt.Sprintf("pod/%s container/%s", pod.Name, ctr.Name),
					Detail:      "securityContext.privileged is true",
					Remediation: "Remove privileged: true. Use specific capabilities (securityContext.capabilities.add) instead.",
				})
			}
		}
	}
	if len(results) == 0 {
		return pass(chk.ID(), chk.Name(), chk.Category()), nil
	}
	return results, nil
}

// --- sec-003: Secrets exposed as environment variables ---

type SecretsAsEnvVars struct{}

func (SecretsAsEnvVars) ID() string              { return "sec-003" }
func (SecretsAsEnvVars) Name() string             { return "Secrets exposed as environment variables" }
func (SecretsAsEnvVars) Category() types.Category { return types.CategorySecurity }

func (chk SecretsAsEnvVars) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	pods, err := client.Typed.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: list pods: %w", chk.ID(), err)
	}

	var results []types.CheckResult
	for _, pod := range pods.Items {
		for _, ctr := range pod.Spec.Containers {
			for _, env := range ctr.Env {
				if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
					results = append(results, types.CheckResult{
						ID:          chk.ID(),
						Category:    chk.Category(),
						Name:        chk.Name(),
						Severity:    types.SeverityWarning,
						Namespace:   pod.Namespace,
						Resource:    fmt.Sprintf("pod/%s container/%s", pod.Name, ctr.Name),
						Detail:      fmt.Sprintf("secret %q injected as env var %q", env.ValueFrom.SecretKeyRef.Name, env.Name),
						Remediation: "Mount secrets as files via volume mounts. Env vars are visible in process listings and debugging output.",
					})
				}
			}
		}
	}
	if len(results) == 0 {
		return pass(chk.ID(), chk.Name(), chk.Category()), nil
	}
	return results, nil
}

// --- sec-004: Missing NetworkPolicies ---

type MissingNetworkPolicies struct{}

func (MissingNetworkPolicies) ID() string              { return "sec-004" }
func (MissingNetworkPolicies) Name() string             { return "Missing NetworkPolicies" }
func (MissingNetworkPolicies) Category() types.Category { return types.CategorySecurity }

func (chk MissingNetworkPolicies) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	systemNS := map[string]bool{
		"kube-system": true, "kube-public": true, "kube-node-lease": true,
	}

	// Build the list of namespaces to inspect.
	var namespaceNames []string
	if namespace != "" {
		namespaceNames = []string{namespace}
	} else {
		nsList, err := client.Typed.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("%s: list namespaces: %w", chk.ID(), err)
		}
		for _, ns := range nsList.Items {
			if !systemNS[ns.Name] {
				namespaceNames = append(namespaceNames, ns.Name)
			}
		}
	}

	var results []types.CheckResult
	for _, ns := range namespaceNames {
		policies, err := client.Typed.NetworkingV1().NetworkPolicies(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}
		if len(policies.Items) == 0 {
			results = append(results, types.CheckResult{
				ID:          chk.ID(),
				Category:    chk.Category(),
				Name:        chk.Name(),
				Severity:    types.SeverityWarning,
				Namespace:   ns,
				Resource:    fmt.Sprintf("namespace/%s", ns),
				Detail:      "no NetworkPolicy resources found — all pods can communicate freely",
				Remediation: "Add a default-deny NetworkPolicy and explicit allow rules for required pod-to-pod traffic.",
			})
		}
	}
	if len(results) == 0 {
		return pass(chk.ID(), chk.Name(), chk.Category()), nil
	}
	return results, nil
}

// --- sec-005: Public image registries ---

type PublicImageRegistries struct{}

func (PublicImageRegistries) ID() string              { return "sec-005" }
func (PublicImageRegistries) Name() string             { return "Pods using public image registries" }
func (PublicImageRegistries) Category() types.Category { return types.CategorySecurity }

func (chk PublicImageRegistries) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	// TODO: list pods, check image prefix against known public registries
	// (docker.io, ghcr.io, quay.io, public.ecr.aws, registry.k8s.io)
	return []types.CheckResult{}, nil
}

// --- sec-006: RBAC wildcard permissions ---

type RBACWildcardPermissions struct{}

func (RBACWildcardPermissions) ID() string              { return "sec-006" }
func (RBACWildcardPermissions) Name() string             { return "RBAC wildcard permissions" }
func (RBACWildcardPermissions) Category() types.Category { return types.CategorySecurity }

func (chk RBACWildcardPermissions) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	// TODO: list ClusterRoles and Roles, flag any rules with "*" verbs or resources
	return []types.CheckResult{}, nil
}

// --- sec-007: hostNetwork / hostPID ---

type HostNetworkHostPID struct{}

func (HostNetworkHostPID) ID() string              { return "sec-007" }
func (HostNetworkHostPID) Name() string             { return "hostNetwork or hostPID enabled" }
func (HostNetworkHostPID) Category() types.Category { return types.CategorySecurity }

func (chk HostNetworkHostPID) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	pods, err := client.Typed.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: list pods: %w", chk.ID(), err)
	}

	var results []types.CheckResult
	for _, pod := range pods.Items {
		if !pod.Spec.HostNetwork && !pod.Spec.HostPID {
			continue
		}
		detail := ""
		if pod.Spec.HostNetwork {
			detail += "hostNetwork=true "
		}
		if pod.Spec.HostPID {
			detail += "hostPID=true"
		}
		results = append(results, types.CheckResult{
			ID:          chk.ID(),
			Category:    chk.Category(),
			Name:        chk.Name(),
			Severity:    types.SeverityCritical,
			Namespace:   pod.Namespace,
			Resource:    fmt.Sprintf("pod/%s", pod.Name),
			Detail:      detail,
			Remediation: "Remove hostNetwork/hostPID unless the pod is node-level infrastructure (e.g. CNI plugin, DaemonSet).",
		})
	}
	if len(results) == 0 {
		return pass(chk.ID(), chk.Name(), chk.Category()), nil
	}
	return results, nil
}
