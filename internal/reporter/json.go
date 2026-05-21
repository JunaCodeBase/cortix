package reporter

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/JunaCodeBase/cortix/pkg/types"
)

// PrintJSON encodes the full ScanResult as indented JSON to w.
// Use PrintGroupedJSON for grouped output (preferred for deep scans).
func PrintJSON(w io.Writer, result *types.ScanResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		return fmt.Errorf("reporter: json encode: %w", err)
	}
	return nil
}

// affectedEntry is one resource that failed a check.
type affectedEntry struct {
	Namespace string `json:"namespace,omitempty"`
	Resource  string `json:"resource,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

// groupedCheck is one check ID's summary with all affected resources collapsed.
type groupedCheck struct {
	Name        string          `json:"name"`
	Category    types.Category  `json:"category"`
	Severity    types.Severity  `json:"severity"`
	Passed      bool            `json:"passed"`
	Remediation string          `json:"remediation,omitempty"`
	Affected    []affectedEntry `json:"affected,omitempty"`
}

// groupedOutput is the top-level grouped JSON document.
type groupedOutput struct {
	ClusterName string                   `json:"cluster_name"`
	ScannedAt   string                   `json:"scanned_at"`
	Mode        types.ScanMode           `json:"mode"`
	Score       *types.Score             `json:"score,omitempty"`
	Checks      map[string]*groupedCheck `json:"checks"`
}

// PrintGroupedJSON writes a grouped JSON document where each check ID appears
// exactly once, with all affected resources collected into an "affected" array.
// It supersedes PrintJSON for deep-scan output.
func PrintGroupedJSON(w io.Writer, result *types.ScanResult) error {
	out := &groupedOutput{
		ClusterName: result.ClusterName,
		ScannedAt:   result.ScannedAt.UTC().Format("2006-01-02T15:04:05Z"),
		Mode:        result.Mode,
		Score:       result.Score,
		Checks:      make(map[string]*groupedCheck),
	}

	for _, cat := range result.Categories {
		for _, chk := range cat.Checks {
			grp, exists := out.Checks[chk.ID]
			if !exists {
				grp = &groupedCheck{
					Name:        chk.Name,
					Category:    chk.Category,
					Severity:    chk.Severity,
					Passed:      chk.Passed,
					Remediation: chk.Remediation,
				}
				out.Checks[chk.ID] = grp
			}

			// Merge: if any result is not passed, mark the group as not passed.
			if !chk.Passed {
				grp.Passed = false
				// Upgrade severity if this result is worse.
				if severityRankJSON(chk.Severity) > severityRankJSON(grp.Severity) {
					grp.Severity = chk.Severity
				}
				if chk.Remediation != "" && grp.Remediation == "" {
					grp.Remediation = chk.Remediation
				}
				grp.Affected = append(grp.Affected, affectedEntry{
					Namespace: chk.Namespace,
					Resource:  chk.Resource,
					Detail:    chk.Detail,
				})
			}
		}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("reporter: grouped json encode: %w", err)
	}
	return nil
}

func severityRankJSON(s types.Severity) int {
	switch s {
	case types.SeverityCritical:
		return 3
	case types.SeverityWarning:
		return 2
	case types.SeverityImprovement:
		return 1
	default:
		return 0
	}
}
