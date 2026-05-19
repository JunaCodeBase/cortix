package checks

import (
	"context"

	"github.com/JunaCodeBase/cortix/internal/k8s"
	"github.com/JunaCodeBase/cortix/pkg/types"
)

// Check is implemented by every deep-scan check.
// Each check is responsible for one specific finding (e.g. "pods running as root").
// Run must be safe to call concurrently across different checks.
type Check interface {
	// ID returns a stable, unique identifier (e.g. "sec-001").
	ID() string
	// Name returns the human-readable check title.
	Name() string
	// Category returns which of the five scan dimensions this check belongs to.
	Category() types.Category
	// Run queries the cluster and returns one CheckResult per affected resource.
	// An empty slice (not nil) means the check passed.
	// namespace is "" when scanning cluster-wide.
	Run(ctx context.Context, client *k8s.Client, namespace string) ([]types.CheckResult, error)
}

// pass is a helper that returns a single PASS result for a check that found no issues.
func pass(id, name string, cat types.Category) []types.CheckResult {
	return []types.CheckResult{{
		ID:       id,
		Category: cat,
		Name:     name,
		Severity: types.SeverityPass,
		Passed:   true,
	}}
}
