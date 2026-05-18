# Cortix CLI — Command Reference

---

## cortix help

Print all commands and flags to the terminal.

| Flag | Default | Description |
|------|---------|-------------|
| _(none)_ | | |

**Example**

```bash
cortix help
```

---

## cortix scan

Scan a Kubernetes cluster for missing or misconfigured infrastructure.

Default: quick mode — 7 observability tool detectors.

| Flag | Default | Description |
|------|---------|-------------|
| `--kubeconfig` | `$KUBECONFIG` or `~/.kube/config` | Path to kubeconfig file |
| `--context` | current context | Kubeconfig context to use |
| `--namespace` | all namespaces | Scope scan to a single namespace |
| `--deep` | false | Run full 5-category deep scan (100+ checks) |
| `--category` | all | Run only one category: `security` \| `reliability` \| `observability` \| `cost` \| `operations` |
| `--output`, `-o` | `text` | Output format: `text` \| `json` \| `html` |
| `--verbose` | false | Show IMPROVEMENT-level results in addition to CRITICAL and WARNING |
| `--show-healthy` | false | Include passing checks in output |

**Examples**

```bash
# Quick scan — default kubeconfig
cortix scan

# Quick scan — specific context
cortix scan --context staging

# Deep scan — all categories
cortix scan --deep --kubeconfig ~/.kube/config

# Deep scan — security only
cortix scan --deep --category security

# Deep scan — single namespace
cortix scan --deep --namespace production

# JSON output
cortix scan --output json

# HTML report to file
cortix scan --deep --output html > report.html

# Show all checks including passing ones
cortix scan --deep --verbose --show-healthy
```

---

## cortix export

Export a live cluster to a clean, production-ready IaC git repository.

All secrets are sanitized — real values are **never** written to disk.
Every secret value is replaced with `PLACEHOLDER_REPLACE_ME`.

| Flag | Default | Description |
|------|---------|-------------|
| `--kubeconfig` | `$KUBECONFIG` or `~/.kube/config` | Path to kubeconfig file |
| `--context` | current context | Kubeconfig context to use |
| `--namespace` | all non-system namespaces | Export only this namespace |
| `--output`, `-o` | `./cortix-export` | Directory to write exported files |
| `--format` | `kustomize` | Output format: `kustomize` \| `helm` \| `external-secrets` |
| `--add-best-practices` | false | Enrich exported Deployments with rolling update strategy, labels |
| `--dry-run` | false | Preview what would be exported — no files written |

**Examples**

```bash
# Export all non-system namespaces — default kubeconfig
cortix export

# Dry run — preview only, nothing written
cortix export --dry-run

# Export single namespace
cortix export --namespace production

# Export to a specific directory
cortix export --output ./my-cluster-backup

# Export with best-practice enrichment
cortix export --add-best-practices

# Export specific context to custom path
cortix export --context prod-eks --output ./prod-export --add-best-practices
```

**Output structure**

```
cortix-export/
├── README.md
├── kustomization.yaml
├── apply.sh
├── WARNINGS.md
├── HELM_RELEASES.md
├── cluster/
│   ├── namespaces/
│   ├── clusterroles/
│   ├── clusterrolebindings/
│   └── storageclasses/
└── namespaces/<name>/
    ├── deployments/
    ├── statefulsets/
    ├── services/
    ├── ingresses/
    ├── configmaps/
    ├── secrets/          ← all values are PLACEHOLDER_REPLACE_ME
    ├── serviceaccounts/
    ├── hpas/
    └── helm/             ← values.yaml stubs for Helm-managed releases
```
