package export

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ApplyBestPractices enriches obj with production-ready defaults that are
// missing from the live cluster. It never overwrites values that already exist.
// Only Deployments and StatefulSets are enriched today; all other kinds pass through.
func ApplyBestPractices(obj *unstructured.Unstructured) *unstructured.Unstructured {
	switch obj.GetKind() {
	case "Deployment", "StatefulSet":
		ensureRollingUpdate(obj)
		ensureLabels(obj)
	}
	return obj
}

// ensureRollingUpdate sets strategy.type=RollingUpdate for Deployments if missing.
func ensureRollingUpdate(obj *unstructured.Unstructured) {
	if obj.GetKind() != "Deployment" {
		return
	}
	spec, ok := obj.Object["spec"].(map[string]interface{})
	if !ok {
		return
	}
	if _, has := spec["strategy"]; !has {
		spec["strategy"] = map[string]interface{}{
			"type": "RollingUpdate",
			"rollingUpdate": map[string]interface{}{
				"maxSurge":       "25%",
				"maxUnavailable": "25%",
			},
		}
	}
}

// ensureLabels adds a cortix.io/exported=true label to the object metadata
// so exported resources are identifiable in a new cluster.
func ensureLabels(obj *unstructured.Unstructured) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	if _, ok := labels["cortix.io/exported"]; !ok {
		labels["cortix.io/exported"] = "true"
		obj.SetLabels(labels)
	}
}
