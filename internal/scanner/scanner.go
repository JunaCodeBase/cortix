package scanner

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/JunaCodeBase/cortix/internal/checks"
	"github.com/JunaCodeBase/cortix/internal/k8s"
	"github.com/JunaCodeBase/cortix/internal/scoring"
	"github.com/JunaCodeBase/cortix/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Detector is implemented by each quick-scan tool detector.
type Detector interface {
	Name() string
	Detect(ctx context.Context, client *k8s.Client) (types.DetectedTool, error)
}

// Scanner runs quick or deep scans depending on ScanOptions.
type Scanner struct {
	client    *k8s.Client
	detectors []Detector
}

// New creates a Scanner. detectors are used for quick mode only.
func New(client *k8s.Client, detectors []Detector) *Scanner {
	return &Scanner{client: client, detectors: detectors}
}

// Run selects quick or deep mode and returns the aggregated ScanResult.
func (s *Scanner) Run(ctx context.Context, opts types.ScanOptions) (*types.ScanResult, error) {
	if opts.Deep || opts.Category != "" {
		return s.runDeep(ctx, opts)
	}
	return s.runQuick(ctx)
}

// runQuick runs the 7 observability detectors and returns a quick-mode ScanResult.
func (s *Scanner) runQuick(ctx context.Context) (*types.ScanResult, error) {
	result := &types.ScanResult{
		ClusterName: s.client.ClusterName,
		Context:     s.client.ClusterName,
		ScannedAt:   time.Now().UTC(),
		Mode:        types.ScanModeQuick,
		Found:       []types.DetectedTool{},
		Missing:     []types.DetectedTool{},
	}

	for _, d := range s.detectors {
		tool, err := d.Detect(ctx, s.client)
		if err != nil {
			slog.Warn("detector failed", "tool", d.Name(), "err", err)
			tool = types.DetectedTool{Name: d.Name(), Status: types.ToolMissing}
		}
		if tool.Status == types.ToolFound {
			result.Found = append(result.Found, tool)
		} else {
			result.Missing = append(result.Missing, tool)
		}
	}

	return result, nil
}

// runDeep dispatches one goroutine per category, aggregates CheckResults,
// scores each category, then computes the weighted overall score.
func (s *Scanner) runDeep(ctx context.Context, opts types.ScanOptions) (*types.ScanResult, error) {
	result := &types.ScanResult{
		ClusterName: s.client.ClusterName,
		Context:     s.client.ClusterName,
		ScannedAt:   time.Now().UTC(),
		Mode:        types.ScanModeDeep,
	}

	categoryChecks := buildCategoryMap()

	// Filter to a single category if --category is set.
	if opts.Category != "" {
		cat := types.Category(opts.Category)
		chks, ok := categoryChecks[cat]
		if !ok {
			return nil, fmt.Errorf("scanner: unknown category %q", opts.Category)
		}
		categoryChecks = map[types.Category][]checks.Check{cat: chks}
	}

	type catResult struct {
		cat    types.Category
		checks []types.CheckResult
		err    error
	}

	ch := make(chan catResult, len(categoryChecks))
	var wg sync.WaitGroup

	for cat, chks := range categoryChecks {
		wg.Add(1)
		go func(cat types.Category, chks []checks.Check) {
			defer wg.Done()
			var allResults []types.CheckResult
			for _, chk := range chks {
				res, err := chk.Run(ctx, s.client, opts.Namespace)
				if err != nil {
					slog.Warn("check failed", "id", chk.ID(), "err", err)
					continue
				}
				allResults = append(allResults, res...)
			}
			ch <- catResult{cat: cat, checks: allResults}
		}(cat, chks)
	}

	wg.Wait()
	close(ch)

	// Collect results in canonical order.
	resultsByCategory := make(map[types.Category][]types.CheckResult)
	for r := range ch {
		if r.err != nil {
			slog.Warn("category failed", "category", r.cat, "err", r.err)
			continue
		}
		resultsByCategory[r.cat] = r.checks
	}

	for _, cat := range types.AllCategories() {
		chkResults, ok := resultsByCategory[cat]
		if !ok {
			continue
		}
		cr := types.CategoryResult{
			Category: cat,
			Score:    scoring.ScoreCategory(chkResults),
			Checks:   chkResults,
		}
		scoring.CountsBySeverity(&cr)
		result.Categories = append(result.Categories, cr)
	}

	result.Score = scoring.Calculate(result)
	return result, nil
}

// buildCategoryMap returns all checks grouped by category.
func buildCategoryMap() map[types.Category][]checks.Check {
	return map[types.Category][]checks.Check{
		types.CategorySecurity:      checks.SecurityChecks(),
		types.CategoryReliability:   checks.ReliabilityChecks(),
		types.CategoryObservability: checks.ObservabilityChecks(),
		types.CategoryCost:          checks.CostChecks(),
		types.CategoryOperations:    checks.OperationsChecks(),
	}
}

// clusterName attempts to resolve the cluster name from node labels as a fallback.
// The primary source is k8s.Client.ClusterName (from kubeconfig context).
func clusterNameFromNodes(ctx context.Context, client *k8s.Client) string {
	nodes, err := client.Typed.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil || len(nodes.Items) == 0 {
		return "unknown"
	}
	if v, ok := nodes.Items[0].Labels["alpha.eksctl.io/cluster-name"]; ok {
		return v
	}
	if v, ok := nodes.Items[0].Labels["cloud.google.com/gke-cluster-name"]; ok {
		return v
	}
	return "unknown"
}
