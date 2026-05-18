package export

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	helmReleaseAnnotation   = "meta.helm.sh/release-name"
	helmNamespaceAnnotation = "meta.helm.sh/release-namespace"
)

// HelmRelease represents a detected Helm-managed release in the cluster.
type HelmRelease struct {
	Name      string
	Namespace string
	Resources []string // resource kinds managed by this release
}

// IsHelmManaged returns true if obj was deployed via Helm.
func IsHelmManaged(obj *unstructured.Unstructured) bool {
	annotations := obj.GetAnnotations()
	_, ok := annotations[helmReleaseAnnotation]
	return ok
}

// ExtractHelmRelease returns a HelmRelease descriptor from a Helm-managed object.
func ExtractHelmRelease(obj *unstructured.Unstructured) HelmRelease {
	annotations := obj.GetAnnotations()
	name := annotations[helmReleaseAnnotation]
	ns := annotations[helmNamespaceAnnotation]
	if ns == "" {
		ns = obj.GetNamespace()
	}
	return HelmRelease{
		Name:      name,
		Namespace: ns,
		Resources: []string{obj.GetKind()},
	}
}
