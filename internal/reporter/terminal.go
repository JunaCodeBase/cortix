package reporter

import (
	"fmt"
	"io"
	"strings"

	"github.com/JunaCodeBase/cortix/pkg/types"
	"github.com/fatih/color"
)

var (
	clrCritical    = color.New(color.FgRed, color.Bold)
	clrWarning     = color.New(color.FgYellow, color.Bold)
	clrPass        = color.New(color.FgGreen, color.Bold)
	clrImprovement = color.New(color.FgCyan)
	clrBold        = color.New(color.Bold)
	clrDim         = color.New(color.Faint)
	clrScore       = color.New(color.FgWhite, color.Bold)
)

// PrintTerminal writes the full scan result to w with ANSI colors.
// By default only CRITICAL and WARNING results are shown.
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
	clrBold.Fprintf(w, "  Cluster  : ")
	fmt.Fprintf(w, "%s\n", result.ClusterName)
	clrBold.Fprintf(w, "  Scanned  : ")
	fmt.Fprintf(w, "%s\n", result.ScannedAt.Format("2006-01-02 15:04:05 UTC"))
	clrBold.Fprintf(w, "  Mode     : ")
	fmt.Fprintf(w, "%s\n", result.Mode)
	fmt.Fprintf(w, "\n")
}

func printQuickResult(w io.Writer, result *types.ScanResult) {
	clrBold.Fprintf(w, "  Found (%d)\n", len(result.Found))
	for _, t := range result.Found {
		ver := ""
		if t.Version != "" {
			ver = " v" + t.Version
		}
		clrPass.Fprintf(w, "    %s", iconOK)
		fmt.Fprintf(w, " %s%s  (%s)\n", t.Name, ver, t.Namespace)
	}

	fmt.Fprintf(w, "\n")
	clrBold.Fprintf(w, "  Missing (%d)\n", len(result.Missing))
	for _, t := range result.Missing {
		clrCritical.Fprintf(w, "    %s", iconError)
		fmt.Fprintf(w, " %s\n", t.Name)
	}

	if result.HealthScore > 0 {
		fmt.Fprintf(w, "\n")
		printDivider(w)
		clrScore.Fprintf(w, "  Health Score: %d/10\n", result.HealthScore)
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
	if delta >= 0 {
		deltaStr = clrPass.Sprint(deltaStr + " ▲")
	} else {
		deltaStr = clrCritical.Sprint(deltaStr + " ▼")
	}

	clrBold.Fprintf(w, "  %s %s", categoryIcon(cat.Category), strings.ToUpper(string(cat.Category)))
	fmt.Fprintf(w, "  score: ")
	clrScore.Fprintf(w, "%d/100", cat.Score)
	fmt.Fprintf(w, "  industry avg: %d  delta: %s\n", cat.IndustryAvg, deltaStr)

	for _, chk := range cat.Checks {
		if !shouldShow(chk.Severity, opts) {
			continue
		}

		prefix := "      "
		iconStr, clr := iconAndColor(chk.Severity)

		clr.Fprintf(w, "%s%s", prefix, iconStr)
		fmt.Fprintf(w, " %s", chk.Name)
		if chk.Resource != "" {
			clrDim.Fprintf(w, " — %s", chk.Resource)
		}
		fmt.Fprintln(w)

		if !chk.Passed && chk.Detail != "" {
			clrDim.Fprintf(w, "%s    %s\n", prefix, chk.Detail)
		}
		if !chk.Passed && chk.Remediation != "" {
			clrDim.Fprintf(w, "%s    Fix: ", prefix)
			fmt.Fprintf(w, "%s\n", chk.Remediation)
		}
	}
	fmt.Fprintln(w)
}

func printScoreBlock(w io.Writer, result *types.ScanResult) {
	s := result.Score
	printDivider(w)

	fmt.Fprintf(w, "  Overall Score : ")
	clrScore.Fprintf(w, "%d/100", s.Overall)
	fmt.Fprintf(w, "  (industry avg: %d  delta: %+d)\n", s.IndustryAvg, s.Delta)
	fmt.Fprintf(w, "  Verdict       : ")
	clrBold.Fprintf(w, "%s\n", s.Verdict)
	fmt.Fprintln(w)

	clrBold.Fprintf(w, "  %-16s  %-10s  %-4s  %s\n", "Category", "Your Score", "Avg", "Delta")
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
		deltaClr := clrPass
		if delta < 0 {
			arrow = "▼"
			deltaClr = clrCritical
		}
		fmt.Fprintf(w, "  %-16s  %-10d  %-4d  ", string(cat), score, avgScore)
		deltaClr.Fprintf(w, "%+d %s\n", delta, arrow)
	}
	printDivider(w)
}

func printCTA(w io.Writer) {
	fmt.Fprintf(w, "\n")
	clrDim.Fprintf(w, "  Run `cortix install` to fix this automatically.\n")
	fmt.Fprintf(w, "\n")
}

func printDivider(w io.Writer) {
	clrDim.Fprintf(w, "  %s\n", strings.Repeat("─", 60))
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

func iconAndColor(sev types.Severity) (string, *color.Color) {
	switch sev {
	case types.SeverityCritical:
		return iconError, clrCritical
	case types.SeverityWarning:
		return iconWarn, clrWarning
	case types.SeverityPass:
		return iconOK, clrPass
	default:
		return iconInfo, clrImprovement
	}
}

func categoryIcon(cat types.Category) string {
	switch cat {
	case types.CategorySecurity:
		return clrCritical.Sprint("[SEC]")
	case types.CategoryReliability:
		return clrWarning.Sprint("[REL]")
	case types.CategoryObservability:
		return clrImprovement.Sprint("[OBS]")
	case types.CategoryCost:
		return clrPass.Sprint("[COST]")
	case types.CategoryOperations:
		return color.New(color.FgBlue, color.Bold).Sprint("[OPS]")
	default:
		return "[?]"
	}
}

const (
	iconOK    = "[+]"
	iconError = "[!]"
	iconWarn  = "[~]"
	iconInfo  = "[ ]"
)
