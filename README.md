# cortix

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**See deep. Fix fast.**

Open-source CLI that connects to any Kubernetes cluster, detects missing or misconfigured infrastructure, and tells you exactly what to run to fix it.

---

## What it does

`cortix scan` gives you an instant health report — what observability tooling is present, what is missing, and what is misconfigured.

`cortix export` reverse-engineers a live cluster into a clean, production-ready IaC git repository. Secrets are always sanitized; Helm releases are exported as values stubs.

```
$ cortix scan

Cortix — Cluster Scanner
Scanning cluster: my-production-cluster
────────────────────────────────────────

[+] metrics-server      found (kube-system)
[+] cert-manager        found (cert-manager)
[!] prometheus          not found
[!] grafana             not found
[!] alertmanager        not found
[!] loki                not found
[~] hpa                 found — no policies configured
[!] ingress-nginx       not found

────────────────────────────────────────
4 critical · 1 warning · 2 healthy

Run `cortix install` to fix this automatically.
```

---

## Install

**Go install**

```bash
go install github.com/JunaCodeBase/cortix/cmd/cortix@latest
```

**Build from source**

```bash
git clone https://github.com/JunaCodeBase/cortix.git
cd cortix

# Linux / macOS
go build -o cortix ./cmd/cortix
./cortix help

# Windows (PowerShell)
go build -o cortix.exe ./cmd/cortix
.\cortix.exe help
```

---

## Commands

| Command | Description |
|---------|-------------|
| `cortix help` | Print all commands, flags, and examples |
| `cortix scan` | Quick scan — 7 observability tool detectors |
| `cortix scan --deep` | Full deep scan — 5 categories, 100+ checks, weighted score |
| `cortix scan --deep --category <cat>` | Deep scan for one category only |
| `cortix scan --output json` | JSON output for CI pipelines |
| `cortix scan --output html` | Shareable HTML health report |
| `cortix scan --verbose` | Include IMPROVEMENT-level results |
| `cortix scan --show-healthy` | Include passing checks in output |
| `cortix export` | Export live cluster to clean IaC YAML |
| `cortix export --dry-run` | Preview export — no files written |

---

## Usage examples

```bash
# Quick scan — default kubeconfig
cortix scan

# Scan a specific context
cortix scan --context staging

# Full deep scan, all categories
cortix scan --deep

# Deep scan — security checks only
cortix scan --deep --category security

# Deep scan — scoped to one namespace
cortix scan --deep --namespace production

# JSON output
cortix scan --output json > report.json

# HTML report
cortix scan --deep --output html > report.html

# Show all checks including passing ones
cortix scan --deep --verbose --show-healthy

# Export all non-system namespaces (dry run first)
cortix export --dry-run
cortix export --output ./my-cluster-backup

# Export one namespace with best-practice enrichment
cortix export --namespace production --add-best-practices --output ./prod-export
```

---

## Scan flags

| Flag | Default | Description |
|------|---------|-------------|
| `--kubeconfig` | `$KUBECONFIG` or `~/.kube/config` | Path to kubeconfig |
| `--context` | current context | Kubeconfig context to use |
| `--namespace` | all namespaces | Scope scan to a single namespace |
| `--deep` | false | Run full deep scan (5 categories, 100+ checks) |
| `--category` | all | `security` \| `reliability` \| `observability` \| `cost` \| `operations` |
| `--output`, `-o` | `text` | `text` \| `json` \| `html` |
| `--verbose` | false | Show IMPROVEMENT results |
| `--show-healthy` | false | Include passing checks in output |

## Export flags

| Flag | Default | Description |
|------|---------|-------------|
| `--kubeconfig` | `$KUBECONFIG` or `~/.kube/config` | Path to kubeconfig |
| `--context` | current context | Kubeconfig context to use |
| `--namespace` | all non-system | Export only this namespace |
| `--output`, `-o` | `./cortix-export` | Output directory |
| `--format` | `kustomize` | `kustomize` \| `helm` \| `external-secrets` |
| `--add-best-practices` | false | Enrich Deployments with rolling update strategy and labels |
| `--dry-run` | false | Preview only — no files written |

---

## What cortix scans for

### Quick scan (default)

| Tool | What is checked |
|------|-----------------|
| Prometheus | Deployment presence, scrape config |
| Grafana | Deployment presence, datasource wiring |
| AlertManager | Deployment presence, receiver config |
| Loki | Deployment presence, log pipeline |
| metrics-server | Deployment presence |
| cert-manager | CRD presence |
| ingress-nginx | Deployment presence |

### Deep scan categories

| Category | Weight | Sample checks |
|----------|--------|---------------|
| Security | 30% | Pods running as root, privileged containers, RBAC wildcards, missing NetworkPolicies |
| Reliability | 25% | Single-replica Deployments, missing liveness/readiness probes, CrashLoopBackOff pods |
| Observability | 20% | Prometheus, Grafana, AlertManager, Loki, metrics-server presence and config |
| Cost | 15% | Missing resource limits/requests, unused namespaces, LoadBalancer overuse |
| Operations | 10% | No rolling update strategy, missing StorageClass, HPA policies, TLS on ingress |

---

## Requirements

- Go 1.21+
- A Kubernetes cluster accessible via kubeconfig or in-cluster config
- Supported: EKS, GKE, AKS, self-hosted, kind, minikube

---

## Project structure

```
cortix/
├── cmd/cortix/          — CLI entrypoint
├── internal/
│   ├── scanner/         — scan orchestrator (quick + deep modes)
│   ├── detector/        — tool detectors (label-selector based)
│   ├── checks/          — deep-scan checks (5 categories)
│   ├── scoring/         — weighted score calculator
│   ├── reporter/        — terminal, JSON, and HTML output
│   ├── export/          — reverse-YAML export engine
│   └── k8s/             — client-go wrappers
└── pkg/types/           — shared types
```

---

## License

MIT — see [LICENSE](LICENSE)
