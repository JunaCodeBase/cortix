package types

import "time"

// ScanMode selects quick vs deep scan.
type ScanMode string

const (
	ScanModeQuick ScanMode = "quick"
	ScanModeDeep  ScanMode = "deep"
)

// Category groups checks into the five scan dimensions.
type Category string

const (
	CategorySecurity      Category = "security"
	CategoryReliability   Category = "reliability"
	CategoryObservability Category = "observability"
	CategoryCost          Category = "cost"
	CategoryOperations    Category = "operations"
)

// AllCategories returns all categories in scoring-weight order.
func AllCategories() []Category {
	return []Category{
		CategorySecurity,
		CategoryReliability,
		CategoryObservability,
		CategoryCost,
		CategoryOperations,
	}
}

// Severity is the urgency level of a single check result.
type Severity string

const (
	SeverityCritical    Severity = "CRITICAL"
	SeverityWarning     Severity = "WARNING"
	SeverityImprovement Severity = "IMPROVEMENT"
	SeverityPass        Severity = "PASS"
)

// ScanOptions carries every flag the scan command accepts.
type ScanOptions struct {
	Deep        bool   // --deep
	Verbose     bool   // --verbose: also show IMPROVEMENT
	ShowHealthy bool   // --show-healthy: also show PASS
	Category    string // --category: run only this category
	Namespace   string // --namespace: scope checks to one namespace
	Output      string // --output: text | json | html
}

// CheckResult is the outcome of a single deep-scan check.
type CheckResult struct {
	ID          string   `json:"id"`                    // e.g. "sec-001"
	Category    Category `json:"category"`
	Name        string   `json:"name"`
	Severity    Severity `json:"severity"`
	Passed      bool     `json:"passed"`
	Namespace   string   `json:"namespace,omitempty"`
	Resource    string   `json:"resource,omitempty"`    // e.g. "deployment/my-app"
	Detail      string   `json:"detail,omitempty"`      // what was observed
	Remediation string   `json:"remediation,omitempty"` // how to fix it
}

// CategoryResult aggregates all check results for one category.
type CategoryResult struct {
	Category         Category      `json:"category"`
	Score            int           `json:"score"`             // 0–100
	IndustryAvg      int           `json:"industry_avg"`
	Checks           []CheckResult `json:"checks"`
	CriticalCount    int           `json:"critical_count"`
	WarningCount     int           `json:"warning_count"`
	ImprovementCount int           `json:"improvement_count"`
	PassingCount     int           `json:"passing_count"`
}

// Score is the final weighted result across all categories.
type Score struct {
	Overall     int              `json:"overall"`      // weighted 0–100
	IndustryAvg int              `json:"industry_avg"` // weighted industry baseline
	Delta       int              `json:"delta"`        // Overall - IndustryAvg
	Verdict     string           `json:"verdict"`
	Breakdown   map[Category]int `json:"breakdown"`    // per-category score 0–100
}

// ScanResult is the top-level output for any scan mode.
type ScanResult struct {
	ClusterName string           `json:"cluster_name"`
	Context     string           `json:"context"`
	ScannedAt   time.Time        `json:"scanned_at"`
	Mode        ScanMode         `json:"mode"`

	// Quick mode fields
	Found       []DetectedTool   `json:"found,omitempty"`
	Missing     []DetectedTool   `json:"missing,omitempty"`
	HealthScore int              `json:"health_score,omitempty"` // 0–10

	// Deep mode fields
	Categories  []CategoryResult `json:"categories,omitempty"`
	Score       *Score           `json:"score,omitempty"`
}

// CategoryResultByName returns the CategoryResult for a given category, or nil.
func (r *ScanResult) CategoryResultByName(c Category) *CategoryResult {
	for i := range r.Categories {
		if r.Categories[i].Category == c {
			return &r.Categories[i]
		}
	}
	return nil
}
