package export

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	// SecretPlaceholder replaces every real secret value in exported files.
	SecretPlaceholder = "PLACEHOLDER_REPLACE_ME"

	// SecretWarningAnnotation is added to every exported secret.
	SecretWarningAnnotation = "cortix.io/secret-warning"

	secretWarningValue = "All values are placeholders. Replace every PLACEHOLDER_REPLACE_ME before applying to any cluster."
)

// SanitizeSecret returns a deep copy of obj with all data values replaced by
// SecretPlaceholder and the cortix.io/secret-warning annotation added.
//
// This function MUST be called for every Secret regardless of type.
// Real values are never written to any file — this rule is non-negotiable.
func SanitizeSecret(obj *unstructured.Unstructured) *unstructured.Unstructured {
	out := Strip(obj) // strip cluster noise first

	// Replace data values (base64-encoded in the API, but we treat them as opaque)
	if data, ok := out.Object["data"].(map[string]interface{}); ok {
		for k := range data {
			data[k] = SecretPlaceholder
		}
	}

	// Replace stringData values (plain-text secrets)
	if sd, ok := out.Object["stringData"].(map[string]interface{}); ok {
		for k := range sd {
			sd[k] = SecretPlaceholder
		}
	}

	// Add the warning annotation so it is visible in any diff tool
	annotations := out.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[SecretWarningAnnotation] = secretWarningValue
	out.SetAnnotations(annotations)

	return out
}
