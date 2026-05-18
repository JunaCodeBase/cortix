package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/JunaDev/cortixlabs/internal/detector"
	"github.com/JunaDev/cortixlabs/internal/export"
	"github.com/JunaDev/cortixlabs/internal/k8s"
	"github.com/JunaDev/cortixlabs/internal/reporter"
	"github.com/JunaDev/cortixlabs/internal/scanner"
	"github.com/JunaDev/cortixlabs/pkg/types"
	"github.com/spf13/cobra"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "cortix",
		Short: "See deep. Fix fast.",
		Long:  "Cortix Labs — scans your Kubernetes cluster and reports what is missing.",
	}
	root.AddCommand(scanCmd())
	root.AddCommand(exportCmd())
	root.AddCommand(helpCmd())
	return root
}

func scanCmd() *cobra.Command {
	var opts types.ScanOptions
	var kubeconfigPath string
	var contextName string

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan a Kubernetes cluster",
		Long: `Scan a Kubernetes cluster for missing or misconfigured infrastructure.

Default mode: quick scan — checks 7 observability tools.
Use --deep for the full engine: 5 categories, 100+ checks, weighted score.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runScan(cmd.Context(), kubeconfigPath, contextName, opts)
		},
	}

	// Connection flags
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "path to kubeconfig (default: $KUBECONFIG or ~/.kube/config)")
	cmd.Flags().StringVar(&contextName, "context", "", "kubeconfig context to use")
	cmd.Flags().StringVar(&opts.Namespace, "namespace", "", "scope scan to a single namespace")

	// Scan mode flags
	cmd.Flags().BoolVar(&opts.Deep, "deep", false, "run full deep scan (5 categories, 100+ checks)")
	cmd.Flags().StringVar(&opts.Category, "category", "", "run only one category: security|reliability|observability|cost|operations")

	// Output flags
	cmd.Flags().StringVarP(&opts.Output, "output", "o", "text", "output format: text|json|html")
	cmd.Flags().BoolVar(&opts.Verbose, "verbose", false, "show IMPROVEMENT results in addition to CRITICAL and WARNING")
	cmd.Flags().BoolVar(&opts.ShowHealthy, "show-healthy", false, "include passing checks in output")

	return cmd
}

func runScan(ctx context.Context, kubeconfigPath, contextName string, opts types.ScanOptions) error {
	slog.Info("connecting to cluster", "kubeconfig", kubeconfigPath, "context", contextName)

	client, err := k8s.NewClient(kubeconfigPath, contextName)
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	detectors := []scanner.Detector{
		detector.Prometheus{},
		detector.Grafana,
		detector.AlertManager,
		detector.Loki,
		detector.MetricsServer,
		detector.CertManager,
		detector.IngressNginx,
	}

	s := scanner.New(client, detectors)

	slog.Info("running scan", "mode", modeLabel(opts), "category", opts.Category)

	result, err := s.Run(ctx, opts)
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	return writeOutput(result, opts)
}

func writeOutput(result *types.ScanResult, opts types.ScanOptions) error {
	switch opts.Output {
	case "json":
		return reporter.PrintJSON(os.Stdout, result)

	case "html":
		filename, err := reporter.WriteHTML(os.Stdout, result)
		if err != nil {
			return err
		}
		slog.Info("report written", "file", filename)
		return nil

	default: // text
		reporter.PrintTerminal(os.Stdout, result, opts)
		return nil
	}
}

func exportCmd() *cobra.Command {
	var opts export.ExportOptions
	var kubeconfigPath string
	var contextName string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export a live cluster to a clean IaC git repository",
		Long: `Export reads every resource from a live cluster, strips cluster-assigned noise,
sanitizes secrets (real values are NEVER written), and writes a production-ready
directory of YAML files that can be committed and reapplied to any cluster.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runExport(cmd.Context(), kubeconfigPath, contextName, opts)
		},
	}

	// Connection flags
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "path to kubeconfig (default: $KUBECONFIG or ~/.kube/config)")
	cmd.Flags().StringVar(&contextName, "context", "", "kubeconfig context to use")
	cmd.Flags().StringVar(&opts.Namespace, "namespace", "", "export only this namespace (default: all non-system namespaces)")

	// Export flags
	cmd.Flags().StringVarP(&opts.OutputDir, "output", "o", "./cortix-export", "directory to write exported files")
	cmd.Flags().StringVar(&opts.Format, "format", "kustomize", "output format: kustomize|helm|external-secrets")
	cmd.Flags().BoolVar(&opts.AddBestPractices, "add-best-practices", false, "enrich exported resources with production defaults")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "print what would be exported without writing files")

	return cmd
}

func runExport(ctx context.Context, kubeconfigPath, contextName string, opts export.ExportOptions) error {
	client, err := k8s.NewClient(kubeconfigPath, contextName)
	if err != nil {
		return fmt.Errorf("export: %w", err)
	}

	exporter := export.New(client)
	result, err := exporter.Run(ctx, opts)
	if err != nil {
		return fmt.Errorf("export: %w", err)
	}

	if opts.DryRun {
		fmt.Printf("Dry run complete — cluster: %s\n", result.ClusterName)
		fmt.Printf("  Namespaces: %d\n", len(result.Namespaces))
		fmt.Printf("  Secrets sanitized: %d\n", result.SecretCount)
		fmt.Printf("  Helm releases detected: %d\n", result.HelmCount)
		if len(result.Warnings) > 0 {
			fmt.Printf("  Warnings: %d\n", len(result.Warnings))
		}
		return nil
	}

	fmt.Printf("Export complete → %s\n", opts.OutputDir)
	fmt.Printf("  Cluster: %s\n", result.ClusterName)
	fmt.Printf("  Namespaces exported: %d\n", len(result.Namespaces))
	fmt.Printf("  Secrets sanitized: %d (all values replaced with PLACEHOLDER_REPLACE_ME)\n", result.SecretCount)
	if result.HelmCount > 0 {
		fmt.Printf("  Helm releases detected: %d — see HELM_RELEASES.md\n", result.HelmCount)
	}
	if len(result.Warnings) > 0 {
		fmt.Printf("  Warnings: %d — see WARNINGS.md\n", len(result.Warnings))
	}
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Replace all PLACEHOLDER_REPLACE_ME values in secrets/")
	fmt.Printf("  2. kubectl apply --dry-run=server -f %s\n", opts.OutputDir)
	fmt.Printf("  3. bash %s/apply.sh\n", opts.OutputDir)
	fmt.Println("\nCortix Labs — cortixlabs.io")
	return nil
}

func helpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "help",
		Short: "Show all commands, flags, and examples",
		Long:  "Print a full reference of every cortix command, its flags, and usage examples.",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Print(`
Cortix — See deep. Fix fast.
Infrastructure intelligence for Kubernetes.

COMMANDS
────────────────────────────────────────────────────────────────────────

  cortix scan                    Quick scan — 7 observability detectors
  cortix scan --deep             Full deep scan — 5 categories, 100+ checks
  cortix scan --deep --category  Deep scan for one category only
  cortix export                  Export live cluster to clean IaC YAML
  cortix help                    Show this help

SCAN FLAGS
────────────────────────────────────────────────────────────────────────

  --kubeconfig  <path>     Path to kubeconfig (default: $KUBECONFIG or ~/.kube/config)
  --context     <name>     Kubeconfig context to use
  --namespace   <ns>       Scope scan to a single namespace
  --deep                   Run full deep scan (5 categories, 100+ checks, weighted score)
  --category    <cat>      Single category: security | reliability | observability | cost | operations
  --output, -o  <fmt>      Output format: text (default) | json | html
  --verbose                Show IMPROVEMENT results in addition to CRITICAL and WARNING
  --show-healthy           Include passing checks in output

EXPORT FLAGS
────────────────────────────────────────────────────────────────────────

  --kubeconfig       <path>   Path to kubeconfig
  --context          <name>   Kubeconfig context
  --namespace        <ns>     Export only this namespace (default: all non-system namespaces)
  --output, -o       <dir>    Output directory (default: ./cortix-export)
  --format           <fmt>    kustomize (default) | helm | external-secrets
  --add-best-practices        Enrich Deployments with rolling update strategy and labels
  --dry-run                   Preview what would be exported — no files written

EXAMPLES
────────────────────────────────────────────────────────────────────────

  # Quick scan with default kubeconfig
  cortix scan

  # Scan a specific context
  cortix scan --context staging

  # Full deep scan, all categories
  cortix scan --deep

  # Deep scan — security checks only
  cortix scan --deep --category security

  # Deep scan — single namespace, verbose
  cortix scan --deep --namespace production --verbose

  # JSON output for CI pipelines
  cortix scan --output json

  # Shareable HTML report
  cortix scan --deep --output html > report.html

  # Export all non-system namespaces (dry run first)
  cortix export --dry-run
  cortix export --output ./my-cluster-backup

  # Export one namespace with best-practice enrichment
  cortix export --namespace production --add-best-practices

DEEP SCAN CATEGORIES
────────────────────────────────────────────────────────────────────────

  Security      (30%)  Pods as root, privileged containers, RBAC wildcards,
                        missing NetworkPolicies, public registries, hostNetwork
  Reliability   (25%)  Single-replica Deployments, missing probes,
                        CrashLoopBackOff, latest image tags, no PDBs
  Observability (20%)  Prometheus, Grafana, AlertManager, Loki, metrics-server,
                        tracing (Jaeger/Tempo)
  Cost          (15%)  Missing resource limits/requests, unused namespaces,
                        missing ResourceQuotas, LoadBalancer overuse
  Operations    (10%)  No rolling update strategy, default namespace in prod,
                        cert-manager, ingress TLS, StorageClass, HPA policies

SCORING
────────────────────────────────────────────────────────────────────────

  Each category scores 0–100. Overall = weighted average.
  CRITICAL checks = 0 pts · WARNING = 50 pts · PASS/IMPROVEMENT = 100 pts
  Industry averages: Security 61 · Reliability 72 · Observability 45
                     Cost 58 · Operations 66 · Overall ~61

────────────────────────────────────────────────────────────────────────
  Run cortix install or visit cortixlabs.io to fix issues automatically.
`)
		},
	}
}

func modeLabel(opts types.ScanOptions) string {
	if opts.Category != "" {
		return "single-category:" + opts.Category
	}
	if opts.Deep {
		return "deep"
	}
	return "quick"
}
