package scoring_test

import (
	"testing"

	"github.com/JunaCodeBase/cortix/internal/scoring"
	"github.com/JunaCodeBase/cortix/pkg/types"
)

func TestScoreCategory(t *testing.T) {
	tests := []struct {
		name    string
		results []types.CheckResult
		want    int
	}{
		{
			name:    "empty results → 0",
			results: nil,
			want:    0,
		},
		{
			name: "all PASS → 100",
			results: []types.CheckResult{
				{ID: "sec-001", Severity: types.SeverityPass, Passed: true},
				{ID: "sec-002", Severity: types.SeverityPass, Passed: true},
				{ID: "sec-003", Severity: types.SeverityPass, Passed: true},
			},
			want: 100,
		},
		{
			name: "all IMPROVEMENT → 100",
			results: []types.CheckResult{
				{ID: "ops-001", Severity: types.SeverityImprovement, Passed: true},
				{ID: "ops-002", Severity: types.SeverityImprovement, Passed: true},
			},
			want: 100,
		},
		{
			name: "all CRITICAL → 0",
			results: []types.CheckResult{
				{ID: "sec-001", Severity: types.SeverityCritical},
				{ID: "sec-002", Severity: types.SeverityCritical},
			},
			want: 0,
		},
		{
			name: "all WARNING → 50",
			results: []types.CheckResult{
				{ID: "rel-001", Severity: types.SeverityWarning},
				{ID: "rel-002", Severity: types.SeverityWarning},
			},
			want: 50,
		},
		{
			name: "one PASS one CRITICAL → 50",
			results: []types.CheckResult{
				{ID: "sec-001", Severity: types.SeverityPass, Passed: true},
				{ID: "sec-002", Severity: types.SeverityCritical},
			},
			want: 50,
		},
		{
			name: "one PASS one WARNING → 75",
			results: []types.CheckResult{
				{ID: "sec-001", Severity: types.SeverityPass, Passed: true},
				{ID: "sec-002", Severity: types.SeverityWarning},
			},
			want: 75,
		},
		{
			name: "duplicate ID: worst severity wins — CRITICAL overrides PASS",
			results: []types.CheckResult{
				{ID: "sec-001", Severity: types.SeverityPass, Passed: true, Resource: "pod/a"},
				{ID: "sec-001", Severity: types.SeverityCritical, Resource: "pod/b"},
			},
			want: 0, // only one unique ID, worst is CRITICAL → 0 pts
		},
		{
			name: "duplicate ID: PASS does not override CRITICAL",
			results: []types.CheckResult{
				{ID: "sec-001", Severity: types.SeverityCritical, Resource: "pod/a"},
				{ID: "sec-001", Severity: types.SeverityPass, Passed: true, Resource: "pod/b"},
			},
			want: 0,
		},
		{
			name: "two checks: one WARNING one CRITICAL → (50+0)/2 = 25",
			results: []types.CheckResult{
				{ID: "sec-001", Severity: types.SeverityWarning},
				{ID: "sec-002", Severity: types.SeverityCritical},
			},
			want: 25,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := scoring.ScoreCategory(tc.results)
			if got != tc.want {
				t.Errorf("ScoreCategory() = %d, want %d", got, tc.want)
			}
		})
	}
}

