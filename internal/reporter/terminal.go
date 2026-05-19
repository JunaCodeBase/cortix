package reporter

import (
	"fmt"
	"io"
	"strings"

	"github.com/JunaCodeBase/cortix/pkg/types"
)

// PrintTerminal writes the full scan result to w using colored terminal output.
// By default only CRITICAL and WARNING results are shown.
// opts controls verbosity and which results are included.
func PrintTerminal(w io.Writer, result *types.ScanResult, opts types.ScanOptions) {
	printHeader(w, result)

	if result.Mode == types.ScanModeQuick {
		printQuickResult(w, result)
	} else {
		printDeepResult(w, result, opts)
	}

	printCTA(w)
}

func printHeader(w io.Writer, result *types.ScanResult) {
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "  Cluster  : %s\n", result.ClusterName)
	fmt.Fprintf(w, "  Scanned  : %s\n", result.ScannedAt.Format("2006-01-02 15:04:05 UTC"))
	fmt.Fprintf(w, "  Mode     : %s\n", result.Mode)
	fmt.Fprintf(w, "\n")
}

func printQuickResult(w io.Writer, result *types.ScanResult) {
	fmt.Fprintf(w, "  Found (%d)\n", len(result.Found))
	for _, t := range result.Found {
		ver := ""
		if t.Version != "" {
			ver = " v" + t.Version
		}
		fmt.Fprintf(w, "    %s %s%s  (%s)\n", iconOK, t.Name, ver, t.Namespace)
	}

	fmt.Fprintf(w, "\n  Missing (%d)\n", len(result.Missing))
	for _, t := range result.Missing {
		fmt.Fprintf(w, "    %s %s\n", iconError, t.Name)
	}

	if result.HealthScore > 0 {
		fmt.Fprintf(w, "\n")
		printDivider(w)
		fmt.Fprintf(w, "  Health Score: %d/10\n", result.HealthScore)
		printDivider(w)
	}
}

func printDeepResult(w io.Writer, result *types.ScanResult, opts types.ScanOptions) {
	for _, cat := range result.Categories {
		printCategory(w, cat, opts)
	}

	if result.Score != nil {
		printScoreBlock(w, result)
	}
}

func printCategory(w io.Writer, cat types.CategoryResult, opts types.ScanOptions) {
	delta := cat.Score - cat.IndustryAvg
	deltaStr := fmt.Sprintf("%+d", delta)

	fmt.Fprintf(w, "  %s %s  score: %d/100  industry avg: %d  delta: %s\n",
		categoryIcon(cat.Category), strings.ToUpper(string(cat.Category)),
		cat.Score, cat.IndustryAvg, deltaStr)

	for _, chk := range cat.Checks {
		if !shouldShow(chk.Severity, opts) {
			continue
		}

		icon := iconFor(chk.Severity)
		prefix := "      "

		fmt.Fprintf(w, "%s%s %s", prefix, icon, chk.Name)
		if chk.Resource != "" {
			fmt.Fprintf(w, " — %s", chk.Resource)
		}
		fmt.Fprintln(w)

		if !chk.Passed && chk.Detail != "" {
			fmt.Fprintf(w, "%s    %s\n", prefix, chk.Detail)
		}
		if !chk.Passed && chk.Remediation != "" {
			fmt.Fprintf(w, "%s    Fix: %s\n", prefix, chk.Remediation)
		}
	}
	fmt.Fprintln(w)
}

func printScoreBlock(w io.Writer, result *types.ScanResult) {
	s := result.Score
	printDivider(w)
	fmt.Fprintf(w, "  Overall Score : %d/100  (industry avg: %d  delta: %+d)\n",
		s.Overall, s.IndustryAvg, s.Delta)
	fmt.Fprintf(w, "  Verdict       : %s\n", s.Verdict)
	fmt.Fprintln(w)

	fmt.Fprintf(w, "  %-16s  %s  %s  %s\n", "Category", "Your Score", "Avg", "Delta")
	fmt.Fprintf(w, "  %s\n", strings.Repeat("-", 52))
	for _, cat := range types.AllCategories() {
		score := s.Breakdown[cat]
		avg := result.CategoryResultByName(cat)
		avgScore := 0
		if avg != nil {
			avgScore = avg.IndustryAvg
		}
		delta := score - avgScore
		arrow := "▲"
		if delta < 0 {
			arrow = "▼"
		}
		fmt.Fprintf(w, "  %-16s  %-10d  %-4d  %+d %s\n",
			string(cat), score, avgScore, delta, arrow)
	}
	printDivider(w)
}

func printCTA(w io.Writer) {
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "  Run `cortix install` to fix this automatically.\n")
	fmt.Fprintf(w, "\n")
}

func printDivider(w io.Writer) {
	fmt.Fprintf(w, "  %s\n", strings.Repeat("─", 60))
}

func shouldShow(sev types.Severity, opts types.ScanOptions) bool {
	switch sev {
	case types.SeverityCritical, types.SeverityWarning:
		return true
	case types.SeverityImprovement:
		return opts.Verbose
	case types.SeverityPass:
		return opts.ShowHealthy
	default:
		return false
	}
}

func iconFor(sev types.Severity) string {
	switch sev {
	case types.SeverityCritical:
		return iconError
	case types.SeverityWarning:
		return iconWarn
	case types.SeverityPass:
		return iconOK
	default:
		return iconInfo
	}
}

func categoryIcon(cat types.Category) string {
	switch cat {
	case types.CategorySecurity:
		return "[SEC]"
	case types.CategoryReliability:
		return "[REL]"
	case types.CategoryObservability:
		return "[OBS]"
	case types.CategoryCost:
		return "[COST]"
	case types.CategoryOperations:
		return "[OPS]"
	default:
		return "[?]"
	}
}

// Terminal symbols — simple ASCII for maximum compatibility.
// A future enhancement can use fatih/color for ANSI colors.
const (
	iconOK    = "[+]"
	iconError = "[!]"
	iconWarn  = "[~]"
	iconInfo  = "[ ]"
)
