package types_test

import (
	"testing"

	"github.com/JunaCodeBase/cortix/pkg/types"
)

func TestExcludeFilter_Matches(t *testing.T) {
	tests := []struct {
		name         string
		filter       types.ExcludeFilter
		resourceType string
		resourceName string
		want         bool
	}{
		// --- Substring match (default) ---
		{
			name:         "substring: value present in name",
			filter:       types.ExcludeFilter{Values: []string{"namespace:kube-system"}},
			resourceType: "namespace",
			resourceName: "kube-system",
			want:         true,
		},
		{
			name:         "substring: partial name match",
			filter:       types.ExcludeFilter{Values: []string{"deployment:nginx"}},
			resourceType: "deployment",
			resourceName: "nginx-ingress",
			want:         true,
		},
		{
			name:         "substring: different resource type — no match",
			filter:       types.ExcludeFilter{Values: []string{"namespace:kube-system"}},
			resourceType: "deployment",
			resourceName: "kube-system",
			want:         false,
		},
		{
			name:         "substring: name not present — no match",
			filter:       types.ExcludeFilter{Values: []string{"pod:redis"}},
			resourceType: "pod",
			resourceName: "nginx-abc",
			want:         false,
		},

		// --- Case-insensitive (-i) ---
		{
			name:         "case-insensitive: uppercase name matches lowercase pattern",
			filter:       types.ExcludeFilter{Values: []string{"namespace:KUBE-SYSTEM"}, CaseInsensitive: true},
			resourceType: "namespace",
			resourceName: "kube-system",
			want:         true,
		},
		{
			name:         "case-insensitive: mixed case both sides",
			filter:       types.ExcludeFilter{Values: []string{"deployment:Nginx"}, CaseInsensitive: true},
			resourceType: "deployment",
			resourceName: "NGINX-ingress",
			want:         true,
		},
		{
			name:         "case-sensitive default: uppercase pattern does not match lowercase name",
			filter:       types.ExcludeFilter{Values: []string{"namespace:KUBE-SYSTEM"}},
			resourceType: "namespace",
			resourceName: "kube-system",
			want:         false,
		},

		// --- Exact match (-e) ---
		{
			name:         "exact: full name matches",
			filter:       types.ExcludeFilter{Values: []string{"deployment:nginx"}, ExactMatch: true},
			resourceType: "deployment",
			resourceName: "nginx",
			want:         true,
		},
		{
			name:         "exact: partial name does not match",
			filter:       types.ExcludeFilter{Values: []string{"deployment:nginx"}, ExactMatch: true},
			resourceType: "deployment",
			resourceName: "nginx-ingress",
			want:         false,
		},
		{
			name:         "exact + case-insensitive: uppercase matches lowercase",
			filter:       types.ExcludeFilter{Values: []string{"pod:Redis"}, ExactMatch: true, CaseInsensitive: true},
			resourceType: "pod",
			resourceName: "redis",
			want:         true,
		},
		{
			name:         "exact + case-sensitive: uppercase does not match lowercase",
			filter:       types.ExcludeFilter{Values: []string{"pod:Redis"}, ExactMatch: true},
			resourceType: "pod",
			resourceName: "redis",
			want:         false,
		},

		// --- Multiple exclude values ---
		{
			name: "multiple values: first matches",
			filter: types.ExcludeFilter{
				Values: []string{"namespace:kube-system", "deployment:nginx"},
			},
			resourceType: "namespace",
			resourceName: "kube-system",
			want:         true,
		},
		{
			name: "multiple values: second matches",
			filter: types.ExcludeFilter{
				Values: []string{"namespace:kube-system", "deployment:nginx"},
			},
			resourceType: "deployment",
			resourceName: "nginx",
			want:         true,
		},
		{
			name: "multiple values: none match",
			filter: types.ExcludeFilter{
				Values: []string{"namespace:kube-system", "deployment:nginx"},
			},
			resourceType: "pod",
			resourceName: "redis",
			want:         false,
		},

		// --- Empty filter ---
		{
			name:         "empty filter: never matches",
			filter:       types.ExcludeFilter{},
			resourceType: "namespace",
			resourceName: "default",
			want:         false,
		},

		// --- Pattern without colon (matches any resource type by name) ---
		{
			name:         "no colon: substring match on name regardless of type",
			filter:       types.ExcludeFilter{Values: []string{"redis"}},
			resourceType: "pod",
			resourceName: "redis-leader",
			want:         true,
		},
		{
			name:         "no colon: no match when name differs",
			filter:       types.ExcludeFilter{Values: []string{"redis"}},
			resourceType: "deployment",
			resourceName: "nginx",
			want:         false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.filter.Matches(tc.resourceType, tc.resourceName)
			if got != tc.want {
				t.Errorf("Matches(%q, %q) = %v, want %v (filter=%+v)",
					tc.resourceType, tc.resourceName, got, tc.want, tc.filter)
			}
		})
	}
}
