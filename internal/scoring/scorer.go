package scoring

import (
	"math"

	"github.com/JunaCodeBase/cortix/pkg/types"
)

// categoryWeight maps each category to its fraction of the overall score.
var categoryWeight = map[types.Category]float64{
	types.CategorySecurity:      0.30,
	types.CategoryReliability:   0.25,
	types.CategoryObservability: 0.20,
	types.CategoryCost:          0.15,
	types.CategoryOperations:    0.10,
}

// industryAvg is the hardcoded v1 baseline. Replace with live data in v2.
var industryAvg = map[types.Category]int{
	types.CategorySecurity:      61,
	types.CategoryReliability:   72,
	types.CategoryObservability: 45,
	types.CategoryCost:          58,
	types.CategoryOperations:    66,
}

// ScoreCategory computes a 0–100 score for one category's check results.
// Each unique check ID is scored by its worst severity:
//
//	PASS / IMPROVEMENT → 100 pts
//	WARNING            → 50 pts
//	CRITICAL           → 0 pts
func ScoreCategory(results []types.CheckResult) int {
	if len(results) == 0 {
		return 0
	}

	// Find worst severity per check ID.
	worstByID := make(map[string]types.Severity)
	for _, r := range results {
		if cur, seen := worstByID[r.ID]; !seen || severityRank(r.Severity) > severityRank(cur) {
			worstByID[r.ID] = r.Severity
		}
	}

	total := 0
	for _, sev := range worstByID {
		total += pointsFor(sev)
	}
	return total / len(worstByID)
}

// IndustryAvgFor returns the hardcoded industry baseline for a category.
func IndustryAvgFor(cat types.Category) int {
	return industryAvg[cat]
}

// Calculate produces the final weighted Score from a completed ScanResult.
func Calculate(result *types.ScanResult) *types.Score {
	breakdown := make(map[types.Category]int, len(result.Categories))
	for _, cat := range result.Categories {
		breakdown[cat.Category] = cat.Score
	}

	overall := 0.0
	industryOverall := 0.0
	for cat, w := range categoryWeight {
		if score, ok := breakdown[cat]; ok {
			overall += float64(score) * w
		}
		industryOverall += float64(industryAvg[cat]) * w
	}

	overallInt := int(math.Round(overall))
	industryInt := int(math.Round(industryOverall))

	return &types.Score{
		Overall:     overallInt,
		IndustryAvg: industryInt,
		Delta:       overallInt - industryInt,
		Verdict:     verdictFor(overallInt),
		Breakdown:   breakdown,
	}
}

// CountsBySeverity fills the critical/warning/improvement/passing counts on a CategoryResult.
func CountsBySeverity(cr *types.CategoryResult) {
	cr.CriticalCount = 0
	cr.WarningCount = 0
	cr.ImprovementCount = 0
	cr.PassingCount = 0
	for _, r := range cr.Checks {
		switch r.Severity {
		case types.SeverityCritical:
			cr.CriticalCount++
		case types.SeverityWarning:
			cr.WarningCount++
		case types.SeverityImprovement:
			cr.ImprovementCount++
		case types.SeverityPass:
			cr.PassingCount++
		}
	}
}

func severityRank(s types.Severity) int {
	switch s {
	case types.SeverityCritical:
		return 3
	case types.SeverityWarning:
		return 2
	case types.SeverityImprovement:
		return 1
	default: // PASS
		return 0
	}
}

func pointsFor(s types.Severity) int {
	switch s {
	case types.SeverityCritical:
		return 0
	case types.SeverityWarning:
		return 50
	default: // PASS, IMPROVEMENT
		return 100
	}
}

func verdictFor(score int) string {
	switch {
	case score >= 90:
		return "excellent — production-grade cluster"
	case score >= 76:
		return "good posture — minor improvements available"
	case score >= 61:
		return "basic posture — gaps should be addressed"
	case score >= 41:
		return "partial coverage — critical gaps remain"
	case score >= 21:
		return "high risk — immediate action required"
	default:
		return "not production-ready"
	}
}
