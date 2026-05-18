# Cortix

**See deep. Fix fast.**

Cortix is an open-source CLI that connects to any Kubernetes cluster, detects missing or misconfigured infrastructure, and tells you exactly what to run to fix it.

---

## What it does

`cortix scan` gives you an instant health report of your cluster — what observability tooling is present, what is missing, and what is misconfigured.

`cortix export` reverse-engineers a live cluster into a clean, production-ready IaC git repository — secrets are always sanitized, Helm releases are noted as values stubs.

```
$ cortix scan --kubeconfig ~/.kube/config

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

Run cortix install or visit cortixlabs.io to fix this automatically.
```

---

## Install

### Go install

```bash
go install github.com/JunaDev/cortixlabs/cmd/cortix@latest
```

### Build from source

```bash
git clone https://github.com/JunaDev/cortixlabs.git
cd cortixlabs/cortex
go build -o cortix ./cmd/cortix
./cortix help
```

---

## Commands

| Command | Description |
|---------|-------------|
| `cortix scan` | Quick scan — detect observability tools (7 detectors) |
| `cortix scan --deep` | Deep scan — 5 categories, 100+ checks, weighted score |
| `cortix scan --category security` | Deep scan for one category only |
| `cortix export` | Export live cluster to clean IaC YAML files |
| `cortix help` | Print all commands and flags |

---

## Usage

```bash
# Quick scan with default kubeconfig
cortix scan

# Scan a specific context
cortix scan --context my-cluster

# Deep scan all categories
cortix scan --deep

# Deep scan — security only
cortix scan --deep --category security

# JSON output
cortix scan --output json

# HTML report
cortix scan --deep --output html > report.html

# Export all namespaces to ./cortix-export
cortix export --kubeconfig ~/.kube/config

# Export single namespace, dry run
cortix export --namespace production --dry-run

# Export with best-practice enrichment
cortix export --add-best-practices --output ./my-export

# Full help
cortix help
```

---

## What cortix scan checks

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
| Observability | 20% | Prometheus + Grafana + AlertManager + Loki + metrics-server presence and config |
| Cost | 15% | Missing resource limits/requests, unused namespaces, LoadBalancer overuse |
| Operations | 10% | No rolling update strategy, missing StorageClass, HPA policies, TLS on ingress |

---

## Requirements

- Go 1.21+
- A Kubernetes cluster (kubeconfig or in-cluster)
- Supported: EKS, GKE, AKS, self-hosted, kind, minikube

---

## Project structure

```
cortex/
├── cmd/cortix/          — CLI entrypoint (main.go)
├── internal/
│   ├── scanner/         — scan orchestrator (quick + deep)
│   ├── detector/        — individual tool detectors
│   ├── checks/          — deep-scan check implementations (5 categories)
│   ├── scoring/         — weighted score calculator
│   ├── reporter/        — terminal, JSON, and HTML output
│   ├── export/          — reverse-YAML export engine
│   └── k8s/             — client-go wrappers
└── pkg/types/           — shared types
```

---

## License

MIT — see [LICENSE](LICENSE)

---

Built by [Cortix Labs](https://cortixlabs.io)
