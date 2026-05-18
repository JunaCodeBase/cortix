package export

import (
	"regexp"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// uuidPattern matches Kubernetes-injected UID values so they can be removed
// from fields where the cluster injects them (e.g. ownerReferences).
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Strip removes all cluster-assigned metadata from obj, returning a deep copy
// that is safe to apply to a fresh cluster. The original is never modified.
func Strip(obj *unstructured.Unstructured) *unstructured.Unstructured {
	out := obj.DeepCopy()
	m := out.Object

	stripMeta(m)
	delete(m, "status")

	switch out.GetKind() {
	case "Service":
		stripServiceSpec(m)
	}

	return out
}

// stripMeta removes the cluster-assigned fields from the metadata map.
func stripMeta(m map[string]interface{}) {
	meta, ok := m["metadata"].(map[string]interface{})
	if !ok {
		return
	}

	for _, field := range []string{
		"resourceVersion",
		"uid",
		"creationTimestamp",
		"generation",
		"managedFields",
		"selfLink",
	} {
		delete(meta, field)
	}

	// Remove specific well-known noisy annotations.
	if annotations, ok := meta["annotations"].(map[string]interface{}); ok {
		for _, key := range []string{
			"kubectl.kubernetes.io/last-applied-configuration",
			"deployment.kubernetes.io/revision",
			"control-plane.alpha.kubernetes.io/leader",
		} {
			delete(annotations, key)
		}
		// Drop any annotation value that is a bare UUID (injected by the cluster).
		for k, v := range annotations {
			if s, ok := v.(string); ok && uuidPattern.MatchString(s) {
				delete(annotations, k)
			}
		}
		if len(annotations) == 0 {
			delete(meta, "annotations")
		}
	}

	// Strip ownerReferences — resources should be standalone in a new cluster.
	delete(meta, "ownerReferences")
}

// stripServiceSpec removes cluster-assigned IP fields from a Service spec.
func stripServiceSpec(m map[string]interface{}) {
	spec, ok := m["spec"].(map[string]interface{})
	if !ok {
		return
	}
	for _, field := range []string{
		"clusterIP",
		"clusterIPs",
		"ipFamilies",
		"ipFamilyPolicy",
		"sessionAffinityConfig",
	} {
		delete(spec, field)
	}
}
