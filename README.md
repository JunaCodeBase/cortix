# cortix

[![License: ELv2](https://img.shields.io/badge/License-Elastic%20v2-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org)

**See deep. Fix fast.**

Open-source Kubernetes intelligence CLI. Scans any cluster, scores it across five categories, and tells you exactly what to run to fix every gap. Works standalone or as a native tool inside Claude Code, Cursor, Codex, and Aider.

---

```
$ cortix scan

Cortix — Cluster Scanner
Scanning cluster: my-production-cluster
────────────────────────────────────────

[+] metrics-server      OK       (kube-system)
[+] cert-manager        OK       (cert-manager)
[!] prometheus          ERROR    not found — no metrics collection
[!] grafana             ERROR    not found
[!] alertmanager        ERROR    not found
[!] loki                ERROR    not found — no log pipeline
[~] ingress-nginx       WARNING  found, no TLS configured

────────────────────────────────────────
Cluster Health Score: 2/10 — not production-ready

Run `cortix install` or visit cortixlabs.io to fix this automatically.
```

---

## Install

**Go install (recommended)**

```bash
go install github.com/JunaCodeBase/cortix/cmd/cortix@latest
```

**Build from source**

```bash
git clone https://github.com/JunaCodeBase/cortix.git
cd cortix

# Linux / macOS
go build -o cortix ./cmd/cortix

# Windows (PowerShell)
go build -o cortix.exe ./cmd/cortix
```

**Requirements:** Go 1.21+ · a Kubernetes cluster reachable via kubeconfig or in-cluster · supports EKS, GKE, AKS, self-hosted, kind, minikube

---

## Use cortix inside your AI coding assistant

Type `/cortix` in your AI assistant and it connects to your live cluster — no copy-pasting YAML, no manual kubectl, no guessing.

**Step 1 — install cortix** (see above)

**Step 2 — register with your assistant:**

| Assistant | Command |
|---|---|
| Claude Code | `cortix install` |
| Cursor | `cortix install --platform cursor` |
| Codex | `cortix install --platform codex` |
| Aider | `cortix install --platform aider` |

**Step 3 — use it:**

```
/cortix what's wrong with my cluster?
/cortix run a security audit
/cortix check my observability setup
/cortix can I export this cluster to IaC?
```

Your assistant calls `cortix_scan`, `cortix_deep_scan`, or `cortix_export_preview` against your live cluster and reasons over the result.

**To remove cortix from your assistant:**

```bash
cortix uninstall                      # Claude Code
cortix uninstall --platform cursor    # Cursor
```

---

## What the AI tools do

When cortix is registered, your AI assistant gains three tools it can call directly:

### `cortix_scan`
Quick scan — checks 7 observability tools (Prometheus, Grafana, AlertManager, Loki, metrics-server, cert-manager, ingress-nginx). Returns presence, severity, business impact text, and the exact Helm command to fix each gap.

### `cortix_deep_scan`
Full scan across 5 categories with weighted scoring:

| Category | Weight | Checks |
|---|---|---|
| Security | 30% | Pods as root, privileged containers, RBAC wildcards, missing NetworkPolicies, public registries, hostNetwork |
| Reliability | 25% | Single-replica Deployments, missing probes, CrashLoopBackOff, latest image tags, no PDBs |
| Observability | 20% | Prometheus, Grafana, AlertManager, Loki, metrics-server, tracing |
| Cost | 15% | Missing resource limits/requests, unused namespaces, ResourceQuota gaps, LoadBalancer overuse |
| Operations | 10% | Rolling update strategy, cert-manager, ingress TLS, StorageClass, HPA policies |

Each category scores 0–100. The weighted overall score gives you a single health number for your cluster.

### `cortix_export_preview`
Dry-run preview of exporting the cluster to a clean IaC git repository — no files written. Shows namespace list, resource counts, Helm releases detected, and warnings.

---

## CLI usage

cortix works fully standalone — no AI assistant required.

### Quick scan
```bash
# Default kubeconfig
cortix scan

# Specific context
cortix scan --context staging

# Scope to one namespace
cortix scan --namespace production

# JSON output (for CI pipelines)
cortix scan --output json > report.json
```

### Deep scan
```bash
# Full scan — all 5 categories
cortix scan deep

# One category only
cortix scan deep --category security
cortix scan deep --category reliability
cortix scan deep --category observability
cortix scan deep --category cost
cortix scan deep --category operations

# Include improvement-level results
cortix scan deep --verbose

# Include passing checks
cortix scan deep --show-healthy

# Shareable HTML report
cortix scan deep --output html > report.html

# Scope to one namespace
cortix scan deep --namespace production --category security
```

### Export
```bash
# Preview what would be exported (no files written)
cortix export --dry-run

# Export all non-system namespaces
cortix export --output ./my-cluster-backup

# Export one namespace
cortix export --namespace production --output ./prod-export

# Export with best-practice enrichment (adds probes, resource limits, labels)
cortix export --namespace production --add-best-practices

# Export as Helm values stubs
cortix export --format helm

# Export with ExternalSecret CRDs instead of placeholder secrets
cortix export --format external-secrets
```

---

## All flags

### `cortix scan` / `cortix scan deep`

| Flag | Default | Description |
|---|---|---|
| `--kubeconfig` | `$KUBECONFIG` or `~/.kube/config` | Path to kubeconfig |
| `--context` | current context | Kubeconfig context to use |
| `--namespace` | all namespaces | Scope scan to a single namespace |
| `--category` | all | `security` \| `reliability` \| `observability` \| `cost` \| `operations` (deep only) |
| `--output`, `-o` | `text` | `text` \| `json` \| `html` |
| `--verbose` | false | Show IMPROVEMENT-level results |
| `--show-healthy` | false | Include passing checks in output |
| `--exclude` | — | Exclude resources: `--exclude namespace:kube-system` (repeatable) |
| `--ignore-case`, `-i` | false | Case-insensitive matching for `--exclude` |
| `--exact`, `-e` | false | Exact match for `--exclude` (default is substring) |

### `cortix export`

| Flag | Default | Description |
|---|---|---|
| `--kubeconfig` | `$KUBECONFIG` or `~/.kube/config` | Path to kubeconfig |
| `--context` | current context | Kubeconfig context to use |
| `--namespace` | all non-system | Export only this namespace |
| `--output`, `-o` | `./cortix-export` | Output directory |
| `--format` | `kustomize` | `kustomize` \| `helm` \| `external-secrets` |
| `--add-best-practices` | false | Enrich Deployments with rolling update strategy, probes, labels |
| `--dry-run` | false | Preview only — no files written |

### `cortix install` / `cortix uninstall`

| Flag | Default | Description |
|---|---|---|
| `--platform` | `claude` | `claude` \| `cursor` \| `codex` \| `aider` |
| `--project-dir` | current directory | Project directory for cursor/codex configs |

---

## Export output structure

```
cortix-export/
├── README.md              auto-generated cluster overview
├── apply.sh               dependency-ordered apply script
├── kustomization.yaml     root kustomize overlay
├── WARNINGS.md            everything needing manual review
├── 00-namespaces/         namespace YAMLs (applied first)
├── 01-cluster/            cluster-wide RBAC, StorageClasses, CRDs
└── <namespace>/           one folder per namespace
    ├── deployments/
    ├── services/
    ├── ingress/
    ├── configmaps/
    ├── secrets/           PLACEHOLDERS ONLY — real values never written
    ├── hpa/
    └── serviceaccounts/
```

**Secrets are always sanitized.** Every secret value is replaced with `PLACEHOLDER_REPLACE_ME`. Real values are never written to any file, ever.

**Helm releases** are detected via annotation and exported as `<release>-values.yaml` stubs with the `helm install` command to redeploy — not as raw manifests.

---

## Project structure

```
cortix/
├── cmd/cortix/          CLI entrypoint
├── internal/
│   ├── scanner/         scan orchestrator (quick + deep modes, parallel categories)
│   ├── detector/        quick-scan detectors (label-selector based)
│   ├── checks/          deep-scan checks (security, reliability, observability, cost, operations)
│   ├── scoring/         weighted scorer (Security 30%, Reliability 25%, Observability 20%, Cost 15%, Operations 10%)
│   ├── reporter/        terminal, JSON, and HTML output formatters
│   ├── export/          reverse-YAML export engine + strip + sanitize
│   ├── install/         cortix install / uninstall for AI assistants
│   ├── mcp/             MCP stdio server (cortix mcp)
│   └── k8s/             client-go wrappers
└── pkg/types/           shared types
```

---

## MCP server (advanced)

cortix ships a built-in MCP server for direct integration with any MCP-compatible tool.

```bash
# Start the MCP server on stdio
cortix mcp
```

Register manually in any MCP config:

```json
{
  "mcpServers": {
    "cortix": {
      "command": "cortix",
      "args": ["mcp"]
    }
  }
}
```

Tools exposed: `cortix_scan` · `cortix_deep_scan` · `cortix_export_preview`

---

## SaaS

The CLI gives you detection and Helm fix commands. **[cortixlabs.io](https://cortixlabs.io)** gives you one-click automated installs, a live cluster dashboard, and team collaboration — no kubectl required.

---

## License

[Elastic License 2.0 (ELv2)](LICENSE)

You are free to use cortix to scan your own infrastructure, modify it, and share it under the same license. You may not offer it as a hosted service or build a commercial product whose primary value comes from cortix's functionality. [Get in touch](https://github.com/JunaCodeBase/cortix/issues) if you want to build on top of it.
