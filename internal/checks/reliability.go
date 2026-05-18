package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/JunaDev/cortixlabs/internal/k8s"
	"github.com/JunaDev/cortixlabs/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ReliabilityChecks returns all reliability checks in priority order.
func ReliabilityChecks() []Check {
	return []Check{
		SingleReplicaDeployments{},
		MissingLivenessProbes{},
		MissingReadinessProbes{},
		NoPodDisruptionBudgets{},
		CrashLoopBackOffPods{},
		HighRestartCountPods{},
		LatestImageTags{},
	}
}

// --- rel-001: Single-replica Deployments ---

type SingleReplicaDeployments struct{}

func (SingleReplicaDeployments) ID() string              { return "rel-001" }
func (SingleReplicaDeployments) Name() string             { return "Single-replica Deployments" }
func (SingleReplicaDeployments) Category() types.Category { return types.CategoryReliability }

func (chk SingleReplicaDeployments) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	deps, err := client.Typed.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: list deployments: %w", chk.ID(), err)
	}

	var results []types.CheckResult
	for _, d := range deps.Items {
		replicas := int32(1)
		if d.Spec.Replicas != nil {
			replicas = *d.Spec.Replicas
		}
		if replicas < 2 {
			results = append(results, types.CheckResult{
				ID:          chk.ID(),
				Category:    chk.Category(),
				Name:        chk.Name(),
				Severity:    types.SeverityWarning,
				Namespace:   d.Namespace,
				Resource:    fmt.Sprintf("deployment/%s", d.Name),
				Detail:      fmt.Sprintf("replicas: %d — single point of failure", replicas),
				Remediation: "Set replicas >= 2 and add a PodDisruptionBudget to maintain availability during node maintenance.",
			})
		}
	}
	if len(results) == 0 {
		return pass(chk.ID(), chk.Name(), chk.Category()), nil
	}
	return results, nil
}

// --- rel-002: Missing liveness probes ---

type MissingLivenessProbes struct{}

func (MissingLivenessProbes) ID() string              { return "rel-002" }
func (MissingLivenessProbes) Name() string             { return "Missing liveness probes" }
func (MissingLivenessProbes) Category() types.Category { return types.CategoryReliability }

func (chk MissingLivenessProbes) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	pods, err := client.Typed.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: list pods: %w", chk.ID(), err)
	}

	var results []types.CheckResult
	for _, pod := range pods.Items {
		for _, ctr := range pod.Spec.Containers {
			if ctr.LivenessProbe == nil {
				results = append(results, types.CheckResult{
					ID:          chk.ID(),
					Category:    chk.Category(),
					Name:        chk.Name(),
					Severity:    types.SeverityWarning,
					Namespace:   pod.Namespace,
					Resource:    fmt.Sprintf("pod/%s container/%s", pod.Name, ctr.Name),
					Detail:      "no livenessProbe configured",
					Remediation: "Add a livenessProbe so Kubernetes can restart stuck containers automatically.",
				})
			}
		}
	}
	if len(results) == 0 {
		return pass(chk.ID(), chk.Name(), chk.Category()), nil
	}
	return results, nil
}

// --- rel-003: Missing readiness probes ---

type MissingReadinessProbes struct{}

func (MissingReadinessProbes) ID() string              { return "rel-003" }
func (MissingReadinessProbes) Name() string             { return "Missing readiness probes" }
func (MissingReadinessProbes) Category() types.Category { return types.CategoryReliability }

func (chk MissingReadinessProbes) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	pods, err := client.Typed.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: list pods: %w", chk.ID(), err)
	}

	var results []types.CheckResult
	for _, pod := range pods.Items {
		for _, ctr := range pod.Spec.Containers {
			if ctr.ReadinessProbe == nil {
				results = append(results, types.CheckResult{
					ID:          chk.ID(),
					Category:    chk.Category(),
					Name:        chk.Name(),
					Severity:    types.SeverityWarning,
					Namespace:   pod.Namespace,
					Resource:    fmt.Sprintf("pod/%s container/%s", pod.Name, ctr.Name),
					Detail:      "no readinessProbe configured",
					Remediation: "Add a readinessProbe so traffic is not sent to pods that are not ready to serve requests.",
				})
			}
		}
	}
	if len(results) == 0 {
		return pass(chk.ID(), chk.Name(), chk.Category()), nil
	}
	return results, nil
}

// --- rel-004: No PodDisruptionBudgets ---

type NoPodDisruptionBudgets struct{}

func (NoPodDisruptionBudgets) ID() string              { return "rel-004" }
func (NoPodDisruptionBudgets) Name() string             { return "No PodDisruptionBudgets" }
func (NoPodDisruptionBudgets) Category() types.Category { return types.CategoryReliability }

func (chk NoPodDisruptionBudgets) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	// TODO: list Deployments with replicas >= 2, check that each has a matching PDB
	return []types.CheckResult{}, nil
}

// --- rel-005: CrashLoopBackOff pods ---

type CrashLoopBackOffPods struct{}

func (CrashLoopBackOffPods) ID() string              { return "rel-005" }
func (CrashLoopBackOffPods) Name() string             { return "CrashLoopBackOff pods" }
func (CrashLoopBackOffPods) Category() types.Category { return types.CategoryReliability }

func (chk CrashLoopBackOffPods) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	pods, err := client.Typed.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: list pods: %w", chk.ID(), err)
	}

	var results []types.CheckResult
	for _, pod := range pods.Items {
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff" {
				results = append(results, types.CheckResult{
					ID:          chk.ID(),
					Category:    chk.Category(),
					Name:        chk.Name(),
					Severity:    types.SeverityCritical,
					Namespace:   pod.Namespace,
					Resource:    fmt.Sprintf("pod/%s container/%s", pod.Name, cs.Name),
					Detail:      fmt.Sprintf("CrashLoopBackOff — restart count: %d", cs.RestartCount),
					Remediation: "Check pod logs: kubectl logs -n " + pod.Namespace + " " + pod.Name + " --previous",
				})
			}
		}
	}
	if len(results) == 0 {
		return pass(chk.ID(), chk.Name(), chk.Category()), nil
	}
	return results, nil
}

// --- rel-006: High restart count pods (> 10) ---

type HighRestartCountPods struct{}

func (HighRestartCountPods) ID() string              { return "rel-006" }
func (HighRestartCountPods) Name() string             { return "High container restart count" }
func (HighRestartCountPods) Category() types.Category { return types.CategoryReliability }

const restartThreshold = 10

func (chk HighRestartCountPods) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	pods, err := client.Typed.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: list pods: %w", chk.ID(), err)
	}

	var results []types.CheckResult
	for _, pod := range pods.Items {
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.RestartCount > restartThreshold {
				results = append(results, types.CheckResult{
					ID:          chk.ID(),
					Category:    chk.Category(),
					Name:        chk.Name(),
					Severity:    types.SeverityWarning,
					Namespace:   pod.Namespace,
					Resource:    fmt.Sprintf("pod/%s container/%s", pod.Name, cs.Name),
					Detail:      fmt.Sprintf("restart count: %d (threshold: %d)", cs.RestartCount, restartThreshold),
					Remediation: "Investigate root cause: kubectl logs -n " + pod.Namespace + " " + pod.Name + " --previous",
				})
			}
		}
	}
	if len(results) == 0 {
		return pass(chk.ID(), chk.Name(), chk.Category()), nil
	}
	return results, nil
}

// --- rel-007: latest image tags ---

type LatestImageTags struct{}

func (LatestImageTags) ID() string              { return "rel-007" }
func (LatestImageTags) Name() string             { return "Containers using :latest image tag" }
func (LatestImageTags) Category() types.Category { return types.CategoryReliability }

func (chk LatestImageTags) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	pods, err := client.Typed.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s: list pods: %w", chk.ID(), err)
	}

	var results []types.CheckResult
	for _, pod := range pods.Items {
		for _, ctr := range pod.Spec.Containers {
			if strings.HasSuffix(ctr.Image, ":latest") || !strings.Contains(ctr.Image, ":") {
				results = append(results, types.CheckResult{
					ID:          chk.ID(),
					Category:    chk.Category(),
					Name:        chk.Name(),
					Severity:    types.SeverityWarning,
					Namespace:   pod.Namespace,
					Resource:    fmt.Sprintf("pod/%s container/%s", pod.Name, ctr.Name),
					Detail:      fmt.Sprintf("image: %s", ctr.Image),
					Remediation: "Pin images to immutable tags (e.g. :v1.2.3 or :<sha256>). The :latest tag causes unpredictable rollouts.",
				})
			}
		}
	}
	if len(results) == 0 {
		return pass(chk.ID(), chk.Name(), chk.Category()), nil
	}
	return results, nil
}
