package reporter_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/JunaCodeBase/cortix/internal/reporter"
	"github.com/JunaCodeBase/cortix/pkg/types"
)

func makeResult(checks ...types.CheckResult) *types.ScanResult {
	return &types.ScanResult{
		ClusterName: "test-cluster",
		Mode:        types.ScanModeDeep,
		ScannedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Categories: []types.CategoryResult{
			{Category: types.CategorySecurity, Checks: checks},
		},
	}
}

func decodeGrouped(t *testing.T, buf *bytes.Buffer) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("JSON decode failed: %v\nbody: %s", err, buf.String())
	}
	return out
}

func checksMap(t *testing.T, out map[string]interface{}) map[string]interface{} {
	t.Helper()
	c, ok := out["checks"].(map[string]interface{})
	if !ok {
		t.Fatalf("no 'checks' map in output")
	}
	return c
}

func TestPrintGroupedJSON_SingleCheckMultipleResources(t *testing.T) {
	result := makeResult(
		types.CheckResult{ID: "sec-001", Name: "Pods Running as Root", Category: types.CategorySecurity, Severity: types.SeverityCritical, Namespace: "prod", Resource: "pod/nginx", Detail: "runAsNonRoot not set", Remediation: "set securityContext.runAsNonRoot=true"},
		types.CheckResult{ID: "sec-001", Name: "Pods Running as Root", Category: types.CategorySecurity, Severity: types.SeverityCritical, Namespace: "prod", Resource: "pod/redis", Detail: "runAsNonRoot not set", Remediation: "set securityContext.runAsNonRoot=true"},
	)

	var buf bytes.Buffer
	if err := reporter.PrintGroupedJSON(&buf, result); err != nil {
		t.Fatalf("PrintGroupedJSON error: %v", err)
	}

	out := decodeGrouped(t, &buf)
	checks := checksMap(t, out)

	if len(checks) != 1 {
		t.Errorf("want 1 check group, got %d", len(checks))
	}

	grp, ok := checks["sec-001"].(map[string]interface{})
	if !ok {
		t.Fatal("sec-001 group missing")
	}

	affected, ok := grp["affected"].([]interface{})
	if !ok {
		t.Fatal("affected array missing from sec-001")
	}
	if len(affected) != 2 {
		t.Errorf("want 2 affected entries, got %d", len(affected))
	}
}

func TestPrintGroupedJSON_TwoDifferentIDs(t *testing.T) {
	result := makeResult(
		types.CheckResult{ID: "sec-001", Name: "Pods Running as Root", Category: types.CategorySecurity, Severity: types.SeverityCritical, Resource: "pod/nginx"},
		types.CheckResult{ID: "sec-002", Name: "Privileged Containers", Category: types.CategorySecurity, Severity: types.SeverityCritical, Resource: "pod/redis"},
	)

	var buf bytes.Buffer
	if err := reporter.PrintGroupedJSON(&buf, result); err != nil {
		t.Fatalf("PrintGroupedJSON error: %v", err)
	}

	checks := checksMap(t, decodeGrouped(t, &buf))
	if len(checks) != 2 {
		t.Errorf("want 2 check groups, got %d", len(checks))
	}
	if _, ok := checks["sec-001"]; !ok {
		t.Error("sec-001 missing from grouped output")
	}
	if _, ok := checks["sec-002"]; !ok {
		t.Error("sec-002 missing from grouped output")
	}
}

func TestPrintGroupedJSON_PassedCheckHasNoAffected(t *testing.T) {
	result := makeResult(
		types.CheckResult{ID: "sec-001", Name: "Pods Running as Root", Category: types.CategorySecurity, Severity: types.SeverityPass, Passed: true},
	)

	var buf bytes.Buffer
	if err := reporter.PrintGroupedJSON(&buf, result); err != nil {
		t.Fatalf("PrintGroupedJSON error: %v", err)
	}

	checks := checksMap(t, decodeGrouped(t, &buf))
	grp, ok := checks["sec-001"].(map[string]interface{})
	if !ok {
		t.Fatal("sec-001 missing")
	}
	if _, hasAffected := grp["affected"]; hasAffected {
		t.Error("passed check should have no 'affected' field")
	}
	if passed, _ := grp["passed"].(bool); !passed {
		t.Error("passed check should have passed=true")
	}
}

func TestPrintGroupedJSON_EmptyResult(t *testing.T) {
	result := &types.ScanResult{
		ClusterName: "empty",
		Mode:        types.ScanModeDeep,
		ScannedAt:   time.Now(),
	}
	var buf bytes.Buffer
	if err := reporter.PrintGroupedJSON(&buf, result); err != nil {
		t.Fatalf("PrintGroupedJSON error: %v", err)
	}
	checks := checksMap(t, decodeGrouped(t, &buf))
	if len(checks) != 0 {
		t.Errorf("empty result should have 0 checks, got %d", len(checks))
	}
}
