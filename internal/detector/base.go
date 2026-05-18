package detector

import (
	"context"
	"fmt"

	"github.com/JunaDev/cortixlabs/internal/k8s"
	"github.com/JunaDev/cortixlabs/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodLabel detects a tool by finding pods that match an app.kubernetes.io label selector.
// Label-based detection is preferred over name matching because Helm release names vary.
type PodLabel struct {
	name     string
	selector string
}

func NewPodLabel(name, selector string) PodLabel {
	return PodLabel{name: name, selector: selector}
}

func (d PodLabel) Name() string { return d.name }

func (d PodLabel) Detect(ctx context.Context, client *k8s.Client) (types.DetectedTool, error) {
	result := types.DetectedTool{Name: d.name, Status: types.ToolMissing}

	pods, err := client.Typed.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		LabelSelector: d.selector,
		Limit:         1,
	})
	if err != nil {
		return result, fmt.Errorf("%s: list pods: %w", d.name, err)
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
