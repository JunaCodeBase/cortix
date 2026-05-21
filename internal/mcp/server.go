package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/JunaCodeBase/cortix/internal/detector"
	"github.com/JunaCodeBase/cortix/internal/export"
	"github.com/JunaCodeBase/cortix/internal/k8s"
	"github.com/JunaCodeBase/cortix/internal/scanner"
	"github.com/JunaCodeBase/cortix/pkg/types"
)

// Serve starts the Cortix MCP server on stdio.
// Claude Code and Cursor connect to this via the mcpServers config in settings.json.
func Serve() error {
	s := server.NewMCPServer("cortix", "1.0.0")

	s.AddTool(scanTool(), handleScan)
	s.AddTool(deepScanTool(), handleDeepScan)
	s.AddTool(exportPreviewTool(), handleExportPreview)

	return server.ServeStdio(s)
}

// --- Tool definitions ---

func scanTool() mcplib.Tool {
	return mcplib.NewTool("cortix_scan",
		mcplib.WithDescription(`Quick-scan a Kubernetes cluster for missing observability tooling.
Checks 7 tools: Prometheus, Grafana, AlertManager, Loki, metrics-server, cert-manager, ingress-nginx.
Returns a JSON report with present/missing tools, severity levels, business impact text, and Helm fix commands.
Use this for a fast first look at a cluster's observability posture.`),
		mcplib.WithString("kubeconfig",
			mcplib.Description("Path to kubeconfig file. Omit to use $KUBECONFIG or ~/.kube/config."),
		),
		mcplib.WithString("context",
			mcplib.Description("Kubeconfig context name. Omit for the current context."),
		),
		mcplib.WithString("namespace",
			mcplib.Description("Scope scan to one namespace. Omit for all namespaces."),
		),
	)
}

func deepScanTool() mcplib.Tool {
	return mcplib.NewTool("cortix_deep_scan",
		mcplib.WithDescription(`Full deep scan across 5 categories with weighted scoring and industry benchmarks.
Categories: Security (30%), Reliability (25%), Observability (20%), Cost (15%), Operations (10%).
Returns per-category scores (0-100), all check results with severity, and delta vs industry averages.
Use this for a full cluster health assessment, security audit, cost review, or reliability analysis.`),
		mcplib.WithString("kubeconfig",
			mcplib.Description("Path to kubeconfig file. Omit to use $KUBECONFIG or ~/.kube/config."),
		),
		mcplib.WithString("context",
			mcplib.Description("Kubeconfig context name. Omit for the current context."),
		),
		mcplib.WithString("namespace",
			mcplib.Description("Scope scan to one namespace. Omit for all namespaces."),
		),
		mcplib.WithString("category",
			mcplib.Description("Run only one category: security | reliability | observability | cost | operations. Omit for all."),
		),
	)
}

func exportPreviewTool() mcplib.Tool {
	return mcplib.NewTool("cortix_export_preview",
		mcplib.WithDescription(`Dry-run preview of exporting a live Kubernetes cluster to a clean IaC git repository.
Shows what would be exported: namespace list, resource counts, Helm releases detected, warnings.
NO files are written to disk. Safe to call at any time.
Use this when the user asks about backing up, exporting, or GitOps-ifying their cluster.`),
		mcplib.WithString("kubeconfig",
			mcplib.Description("Path to kubeconfig file. Omit to use $KUBECONFIG or ~/.kube/config."),
		),
		mcplib.WithString("context",
			mcplib.Description("Kubeconfig context name. Omit for the current context."),
		),
		mcplib.WithString("namespace",
			mcplib.Description("Preview export for one namespace only. Omit for all non-system namespaces."),
		),
	)
}

// --- Handlers ---

func handleScan(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	args := argsMap(req)
	kubeconfigPath, _ := args["kubeconfig"].(string)
	contextName, _ := args["context"].(string)
	namespace, _ := args["namespace"].(string)

	client, err := k8s.NewClient(kubeconfigPath, contextName)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("connect: %s", err)), nil
	}

	s := scanner.New(client, defaultDetectors())
	result, err := s.Run(ctx, types.ScanOptions{Namespace: namespace})
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("scan: %s", err)), nil
	}

	return jsonResult(result)
}

func handleDeepScan(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	args := argsMap(req)
	kubeconfigPath, _ := args["kubeconfig"].(string)
	contextName, _ := args["context"].(string)
	namespace, _ := args["namespace"].(string)
	category, _ := args["category"].(string)

	client, err := k8s.NewClient(kubeconfigPath, contextName)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("connect: %s", err)), nil
	}

	s := scanner.New(client, defaultDetectors())
	result, err := s.Run(ctx, types.ScanOptions{
		Deep:      true,
		Namespace: namespace,
		Category:  category,
	})
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("deep scan: %s", err)), nil
	}

	return jsonResult(result)
}

func handleExportPreview(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	args := argsMap(req)
	kubeconfigPath, _ := args["kubeconfig"].(string)
	contextName, _ := args["context"].(string)
	namespace, _ := args["namespace"].(string)

	client, err := k8s.NewClient(kubeconfigPath, contextName)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("connect: %s", err)), nil
	}

	exporter := export.New(client)
	result, err := exporter.Run(ctx, export.ExportOptions{
		Namespace: namespace,
		DryRun:    true,
		Format:    "kustomize",
	})
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("export preview: %s", err)), nil
	}

	return jsonResult(result)
}

// --- Helpers ---

func defaultDetectors() []scanner.Detector {
	return []scanner.Detector{
		detector.Prometheus{},
		detector.Grafana,
		detector.AlertManager,
		detector.Loki,
		detector.MetricsServer,
		detector.CertManager,
		detector.IngressNginx,
	}
}

func argsMap(req mcplib.CallToolRequest) map[string]any {
	if m, ok := req.Params.Arguments.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func jsonResult(v any) (*mcplib.CallToolResult, error) {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("marshal: %s", err)), nil
	}
	return mcplib.NewToolResultText(string(out)), nil
}
