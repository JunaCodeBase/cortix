package checks

import (
	"context"

	"github.com/JunaDev/cortixlabs/internal/k8s"
	"github.com/JunaDev/cortixlabs/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObservabilityChecks returns all observability checks in priority order.
func ObservabilityChecks() []Check {
	return []Check{
		PrometheusCheck{},
		GrafanaCheck{},
		AlertManagerCheck{},
		LokiCheck{},
		MetricsServerCheck{},
		TracingToolsCheck{},
	}
}

// --- obs-001: Prometheus presence + scrape config ---

type PrometheusCheck struct{}

func (PrometheusCheck) ID() string              { return "obs-001" }
func (PrometheusCheck) Name() string             { return "Prometheus — metrics collection" }
func (PrometheusCheck) Category() types.Category { return types.CategoryObservability }

func (chk PrometheusCheck) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	pods, err := client.Typed.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=prometheus",
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
			Severity:    types.SeverityCritical,
			Passed:      false,
			Detail:      "No metrics collection. You are blind to CPU spikes, memory leaks, and pod crashes in production.",
			Remediation: "helm install prometheus prometheus-community/kube-prometheus-stack -n monitoring --create-namespace",
		}}, nil
	}

	// TODO: check for scrape configs (ConfigMap data or PrometheusRule CRDs)
	return pass(chk.ID(), chk.Name(), chk.Category()), nil
}

// --- obs-002: Grafana presence + datasource ---

type GrafanaCheck struct{}

func (GrafanaCheck) ID() string              { return "obs-002" }
func (GrafanaCheck) Name() string             { return "Grafana — dashboards and visualisation" }
func (GrafanaCheck) Category() types.Category { return types.CategoryObservability }

func (chk GrafanaCheck) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	pods, err := client.Typed.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=grafana",
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
			Severity:    types.SeverityCritical,
			Passed:      false,
			Detail:      "No dashboards. Prometheus metrics exist but cannot be explored or shared with the team.",
			Remediation: "helm install grafana grafana/grafana -n monitoring --create-namespace",
		}}, nil
	}

	// TODO: check for datasource provisioning ConfigMap
	return pass(chk.ID(), chk.Name(), chk.Category()), nil
}

// --- obs-003: AlertManager presence + receivers ---

type AlertManagerCheck struct{}

func (AlertManagerCheck) ID() string              { return "obs-003" }
func (AlertManagerCheck) Name() string             { return "AlertManager — alerting pipeline" }
func (AlertManagerCheck) Category() types.Category { return types.CategoryObservability }

func (chk AlertManagerCheck) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	pods, err := client.Typed.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=alertmanager",
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
			Detail:      "Prometheus found but no alerting configured. Incidents will go unnoticed until someone manually checks dashboards.",
			Remediation: "AlertManager is included in kube-prometheus-stack — re-run the Prometheus Helm install with alerting config.",
		}}, nil
	}

	// TODO: check for receiver configuration in alertmanager.yaml ConfigMap
	return pass(chk.ID(), chk.Name(), chk.Category()), nil
}

// --- obs-004: Loki presence ---

type LokiCheck struct{}

func (LokiCheck) ID() string              { return "obs-004" }
func (LokiCheck) Name() string             { return "Loki — log aggregation" }
func (LokiCheck) Category() types.Category { return types.CategoryObservability }

func (chk LokiCheck) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	pods, err := client.Typed.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=loki",
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
			Severity:    types.SeverityCritical,
			Passed:      false,
			Detail:      "No log aggregation. Pod logs are lost when containers restart. You cannot correlate a Prometheus alert with the logs that caused it.",
			Remediation: "helm install loki grafana/loki-stack -n monitoring --create-namespace",
		}}, nil
	}

	return pass(chk.ID(), chk.Name(), chk.Category()), nil
}

// --- obs-005: metrics-server ---

type MetricsServerCheck struct{}

func (MetricsServerCheck) ID() string              { return "obs-005" }
func (MetricsServerCheck) Name() string             { return "metrics-server — resource usage API" }
func (MetricsServerCheck) Category() types.Category { return types.CategoryObservability }

func (chk MetricsServerCheck) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	pods, err := client.Typed.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=metrics-server",
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
			Detail:      "kubectl top and HPA autoscaling require metrics-server. Without it, horizontal scaling does not work.",
			Remediation: "helm install metrics-server metrics-server/metrics-server -n kube-system",
		}}, nil
	}

	return pass(chk.ID(), chk.Name(), chk.Category()), nil
}

// --- obs-006: Tracing tools (Jaeger / Tempo) ---

type TracingToolsCheck struct{}

func (TracingToolsCheck) ID() string              { return "obs-006" }
func (TracingToolsCheck) Name() string             { return "Distributed tracing (Jaeger / Tempo)" }
func (TracingToolsCheck) Category() types.Category { return types.CategoryObservability }

func (chk TracingToolsCheck) Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error) {
	for _, selector := range []string{
		"app.kubernetes.io/name=jaeger",
		"app.kubernetes.io/name=tempo",
	} {
		pods, err := client.Typed.CoreV1().Pods("").List(ctx, metav1.ListOptions{
			LabelSelector: selector,
			Limit:         1,
		})
		if err == nil && len(pods.Items) > 0 {
			return pass(chk.ID(), chk.Name(), chk.Category()), nil
		}
	}

	return []types.CheckResult{{
		ID:          chk.ID(),
		Category:    chk.Category(),
		Name:        chk.Name(),
		Severity:    types.SeverityImprovement,
		Passed:      false,
		Detail:      "No distributed tracing found. Without tracing, latency spikes across services are hard to diagnose.",
		Remediation: "helm install tempo grafana/tempo -n monitoring --create-namespace",
	}}, nil
}
