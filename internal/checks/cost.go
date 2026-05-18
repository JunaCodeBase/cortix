package checks

import (
	"context"
	"fmt"

	"github.com/JunaDev/cortixlabs/internal/k8s"
	"github.com/JunaDev/cortixlabs/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CostChecks returns all cost efficiency checks in priority order.
func CostChecks() []Check {
	return []Check{
		PodsWithoutResourceLimits{},
		PodsWithoutResourceRequests{},
		MissingNamespaceResourceQuotas{},
		MissingLimitRanges{},
		UnusedNamespaces{},
		LoadBalancerServiceOveruse{},
	}
}

// --- cost-001: Pods without resource limits ---

type PodsWithoutResourceLimits struct{}

func (PodsWithoutResourceLimits) ID() string              { return "cost-001" }
func (PodsWithoutResourceLimits) Name() string             { return "Pods without resource limits" }
func (PodsWithoutResourceLimits) Category() types.Category { return types.CategoryCost }

func (chk PodsWithoutResourceLimits) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	pods, err := client.Typed.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: list pods: %w", chk.ID(), err)
	}

	var results []types.CheckResult
	for _, pod := range pods.Items {
		for _, ctr := range pod.Spec.Containers {
			if ctr.Resources.Limits == nil || len(ctr.Resources.Limits) == 0 {
				results = append(results, types.CheckResult{
					ID:          chk.ID(),
					Category:    chk.Category(),
					Name:        chk.Name(),
					Severity:    types.SeverityWarning,
					Namespace:   pod.Namespace,
					Resource:    fmt.Sprintf("pod/%s container/%s", pod.Name, ctr.Name),
					Detail:      "no CPU or memory limits set — container can consume unlimited node resources",
					Remediation: "Set resources.limits.cpu and resources.limits.memory on every container.",
				})
			}
		}
	}
	if len(results) == 0 {
		return pass(chk.ID(), chk.Name(), chk.Category()), nil
	}
	return results, nil
}

// --- cost-002: Pods without resource requests ---

type PodsWithoutResourceRequests struct{}

func (PodsWithoutResourceRequests) ID() string              { return "cost-002" }
func (PodsWithoutResourceRequests) Name() string             { return "Pods without resource requests" }
func (PodsWithoutResourceRequests) Category() types.Category { return types.CategoryCost }

func (chk PodsWithoutResourceRequests) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	pods, err := client.Typed.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: list pods: %w", chk.ID(), err)
	}

	var results []types.CheckResult
	for _, pod := range pods.Items {
		for _, ctr := range pod.Spec.Containers {
			if ctr.Resources.Requests == nil || len(ctr.Resources.Requests) == 0 {
				results = append(results, types.CheckResult{
					ID:          chk.ID(),
					Category:    chk.Category(),
					Name:        chk.Name(),
					Severity:    types.SeverityWarning,
					Namespace:   pod.Namespace,
					Resource:    fmt.Sprintf("pod/%s container/%s", pod.Name, ctr.Name),
					Detail:      "no resource requests — scheduler cannot make accurate placement decisions",
					Remediation: "Set resources.requests.cpu and resources.requests.memory on every container.",
				})
			}
		}
	}
	if len(results) == 0 {
		return pass(chk.ID(), chk.Name(), chk.Category()), nil
	}
	return results, nil
}

// --- cost-003: Missing namespace ResourceQuotas ---

type MissingNamespaceResourceQuotas struct{}

func (MissingNamespaceResourceQuotas) ID() string              { return "cost-003" }
func (MissingNamespaceResourceQuotas) Name() string             { return "Missing namespace ResourceQuotas" }
func (MissingNamespaceResourceQuotas) Category() types.Category { return types.CategoryCost }

func (chk MissingNamespaceResourceQuotas) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	systemNS := map[string]bool{
		"kube-system": true, "kube-public": true, "kube-node-lease": true,
	}

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
		quotas, err := client.Typed.CoreV1().ResourceQuotas(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}
		if len(quotas.Items) == 0 {
			results = append(results, types.CheckResult{
				ID:          chk.ID(),
				Category:    chk.Category(),
				Name:        chk.Name(),
				Severity:    types.SeverityImprovement,
				Namespace:   ns,
				Resource:    fmt.Sprintf("namespace/%s", ns),
				Detail:      "no ResourceQuota — one runaway workload can exhaust the entire cluster",
				Remediation: "Add a ResourceQuota to cap total CPU, memory, and pod count per namespace.",
			})
		}
	}
	if len(results) == 0 {
		return pass(chk.ID(), chk.Name(), chk.Category()), nil
	}
	return results, nil
}

// --- cost-004: Missing LimitRanges ---

type MissingLimitRanges struct{}

func (MissingLimitRanges) ID() string              { return "cost-004" }
func (MissingLimitRanges) Name() string             { return "Missing LimitRanges" }
func (MissingLimitRanges) Category() types.Category { return types.CategoryCost }

func (chk MissingLimitRanges) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	// TODO: list namespaces, check each for a LimitRange resource
	return []types.CheckResult{}, nil
}

// --- cost-005: Unused / idle namespaces ---

type UnusedNamespaces struct{}

func (UnusedNamespaces) ID() string              { return "cost-005" }
func (UnusedNamespaces) Name() string             { return "Unused or idle namespaces" }
func (UnusedNamespaces) Category() types.Category { return types.CategoryCost }

func (chk UnusedNamespaces) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	// TODO: list namespaces, flag those with zero running pods
	return []types.CheckResult{}, nil
}

// --- cost-006: LoadBalancer service overuse ---

type LoadBalancerServiceOveruse struct{}

func (LoadBalancerServiceOveruse) ID() string              { return "cost-006" }
func (LoadBalancerServiceOveruse) Name() string             { return "LoadBalancer service overuse" }
func (LoadBalancerServiceOveruse) Category() types.Category { return types.CategoryCost }

func (chk LoadBalancerServiceOveruse) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	svcs, err := client.Typed.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: list services: %w", chk.ID(), err)
	}

	var results []types.CheckResult
	for _, svc := range svcs.Items {
		if string(svc.Spec.Type) == "LoadBalancer" {
			results = append(results, types.CheckResult{
				ID:          chk.ID(),
				Category:    chk.Category(),
				Name:        chk.Name(),
				Severity:    types.SeverityImprovement,
				Namespace:   svc.Namespace,
				Resource:    fmt.Sprintf("service/%s", svc.Name),
				Detail:      "type: LoadBalancer provisions a cloud load balancer per service — expensive at scale",
				Remediation: "Consolidate external traffic behind a single ingress-nginx or other ingress controller.",
			})
		}
	}
	if len(results) == 0 {
		return pass(chk.ID(), chk.Name(), chk.Category()), nil
	}
	return results, nil
}
