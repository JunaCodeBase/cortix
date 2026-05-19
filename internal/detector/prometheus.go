package detector

import (
	"context"
	"fmt"

	"github.com/JunaCodeBase/cortix/internal/k8s"
	"github.com/JunaCodeBase/cortix/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Prometheus detects a running Prometheus instance via label selectors.
// Label matching is used instead of name matching because Helm chart names vary.
type Prometheus struct{}

func (Prometheus) Name() string { return "Prometheus" }

func (Prometheus) Detect(ctx context.Context, client *k8s.Client) (types.DetectedTool, error) {
	result := types.DetectedTool{Name: "Prometheus", Status: types.ToolMissing}

	// Prometheus deployments/statefulsets carry this label regardless of release name.
	selector := "app.kubernetes.io/name=prometheus"

	pods, err := client.Typed.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		LabelSelector: selector,
		Limit:         1,
	})
	if err != nil {
		return result, fmt.Errorf("prometheus: list pods: %w", err)
	}

	if len(pods.Items) > 0 {
		result.Status = types.ToolFound
		result.Namespace = pods.Items[0].Namespace
		if v, ok := pods.Items[0].Labels["app.kubernetes.io/version"]; ok {
			result.Version = v
		}
	}

	return result, nil
}
