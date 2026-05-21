# Cortix — Check Parameters

This document describes every check cortix runs during a deep scan (`cortix scan deep`).
Each check has a unique ID, a severity level, and a clear description of what is evaluated
and how to fix it.

---

## Scoring

| Severity | Points | Meaning |
|----------|--------|---------|
| CRITICAL | 0 | Immediate risk — production outage or security breach possible |
| WARNING | 50 | Degraded posture — should be addressed soon |
| IMPROVEMENT | 100 | Best practice not followed — low urgency |
| PASS | 100 | Check passed — no action needed |

**Category score** = average of points across all checks in that category (0–100).

**Overall score** = weighted average:

| Category | Weight | Industry Avg |
|----------|--------|-------------|
| Security | 30% | 61 |
| Reliability | 25% | 72 |
| Observability | 20% | 45 |
| Cost | 15% | 58 |
| Operations | 10% | 66 |
| **Overall** | | **~61** |

---

## Security (weight: 30%)

| ID | Name | Severity | What is checked | Remediation |
|----|------|----------|-----------------|-------------|
| sec-001 | Pods Running as Root | CRITICAL | Lists all pods where `securityContext.runAsNonRoot` is not `true` or `runAsUser` is 0 | Set `securityContext.runAsNonRoot: true` and `runAsUser` to a non-zero UID |
| sec-002 | Privileged Containers | CRITICAL | Lists containers where `securityContext.privileged: true` | Remove `privileged: true`; use specific Linux capabilities instead |
| sec-003 | Secrets Exposed as Env Vars | WARNING | Lists containers that reference Secrets via `env.valueFrom.secretKeyRef` directly | Use mounted secret volumes or an external secrets operator instead |
| sec-004 | Missing NetworkPolicies | WARNING | Checks whether any NetworkPolicy exists in each namespace | Create ingress/egress NetworkPolicies to restrict pod-to-pod traffic |
| sec-005 | Public Image Registries | WARNING | Lists pods pulling images from Docker Hub, ghcr.io, quay.io, or gcr.io without a private mirror | Mirror images to a private registry and update image references |
| sec-006 | RBAC Wildcard Permissions | CRITICAL | Lists ClusterRoles/Roles that contain `"*"` in verbs or resources | Replace wildcard rules with the minimum specific verbs and resources needed |
| sec-007 | hostNetwork / hostPID Usage | CRITICAL | Lists pods with `hostNetwork: true` or `hostPID: true` | Remove `hostNetwork` and `hostPID` unless absolutely required; use NetworkPolicies instead |

---

## Reliability (weight: 25%)

| ID | Name | Severity | What is checked | Remediation |
|----|------|----------|-----------------|-------------|
| rel-001 | Single-Replica Deployments | WARNING | Lists Deployments with `replicas < 2` | Set `replicas: 2` or higher for production workloads |
| rel-002 | Missing Liveness Probes | WARNING | Lists containers with no `livenessProbe` defined | Add a `livenessProbe` (HTTP, TCP, or exec) to each container |
| rel-003 | Missing Readiness Probes | WARNING | Lists containers with no `readinessProbe` defined | Add a `readinessProbe` so traffic is not sent to unready pods |
| rel-004 | No PodDisruptionBudgets | WARNING | Lists Deployments with `replicas >= 2` that have no matching PodDisruptionBudget | Create a PDB with `minAvailable: 1` for each multi-replica Deployment |
| rel-005 | CrashLoopBackOff Pods | CRITICAL | Lists pods currently in `CrashLoopBackOff` state | Check pod logs: `kubectl logs <pod> --previous`; fix the underlying startup failure |
| rel-006 | High Restart Count Pods | WARNING | Lists pods where any container has restarted more than 10 times | Investigate recurring crashes; add liveness probes and resource limits |
| rel-007 | Latest Image Tags | WARNING | Lists containers using the `:latest` image tag | Pin image tags to a specific digest or semantic version |

---

## Observability (weight: 20%)

| ID | Name | Severity | What is checked | Remediation |
|----|------|----------|-----------------|-------------|
| obs-001 | Prometheus Missing | CRITICAL | Checks for a running Prometheus pod (label `app.kubernetes.io/name=prometheus`) | `helm install prometheus prometheus-community/kube-prometheus-stack -n monitoring --create-namespace` |
| obs-002 | Grafana Missing | WARNING | Checks for a running Grafana pod | `helm install grafana grafana/grafana -n monitoring` |
| obs-003 | AlertManager Missing | CRITICAL | Checks for a running AlertManager pod | Installed as part of kube-prometheus-stack; ensure `alertmanager.enabled: true` in Helm values |
| obs-004 | Loki Missing | WARNING | Checks for a running Loki pod | `helm install loki grafana/loki-stack -n monitoring` |
| obs-005 | metrics-server Missing | WARNING | Checks for a running metrics-server pod | `helm install metrics-server metrics-server/metrics-server -n kube-system` |
| obs-006 | No Tracing Tools | IMPROVEMENT | Checks for Jaeger or Tempo pods in the cluster | `helm install jaeger jaegertracing/jaeger -n monitoring` |

---

## Cost Efficiency (weight: 15%)

| ID | Name | Severity | What is checked | Remediation |
|----|------|----------|-----------------|-------------|
| cost-001 | Pods Without Resource Limits | WARNING | Lists containers with no `resources.limits` defined | Add `resources.limits.cpu` and `resources.limits.memory` to every container |
| cost-002 | Pods Without Resource Requests | WARNING | Lists containers with no `resources.requests` defined | Add `resources.requests.cpu` and `resources.requests.memory` to every container |
| cost-003 | Missing Namespace ResourceQuotas | WARNING | Lists namespaces with no ResourceQuota object | Create a ResourceQuota to cap total CPU/memory consumption per namespace |
| cost-004 | Missing LimitRanges | WARNING | Lists namespaces with no LimitRange object | Create a LimitRange to set default requests/limits for containers that omit them |
| cost-005 | Unused Namespaces | WARNING | Lists namespaces with zero running pods | Delete unused namespaces or move workloads out of them |
| cost-006 | LoadBalancer Service Overuse | WARNING | Lists Services of type `LoadBalancer` — each provisions a cloud load balancer | Consolidate behind an Ingress controller; use `ClusterIP` + Ingress instead |

---

## Operations (weight: 10%)

| ID | Name | Severity | What is checked | Remediation |
|----|------|----------|-----------------|-------------|
| ops-001 | No Rolling Update Strategy | WARNING | Lists Deployments missing `strategy.type: RollingUpdate` | Set `strategy: {type: RollingUpdate, rollingUpdate: {maxSurge: "25%", maxUnavailable: "25%"}}` |
| ops-002 | Production Workloads in default Namespace | WARNING | Lists pods running in the `default` namespace | Move workloads to dedicated namespaces with RBAC and quotas |
| ops-003 | cert-manager Missing | IMPROVEMENT | Checks for a running cert-manager pod | `helm install cert-manager jetstack/cert-manager -n cert-manager --set installCRDs=true` |
| ops-004 | Ingress Without TLS | WARNING | Lists Ingress resources with no TLS block | Add a `tls` section referencing a cert-manager-issued certificate |
| ops-005 | No StorageClass Defined | IMPROVEMENT | Checks whether at least one StorageClass exists in the cluster | Install a CSI driver or create a StorageClass for your environment |
| ops-006 | HPA Without Policies | WARNING | Lists HorizontalPodAutoscalers with no `behavior` policies defined | Add `spec.behavior` with `scaleUp` and `scaleDown` policies |

---

## Exclude patterns

Use `--exclude` to skip specific resources during any scan:

```bash
# Skip an entire namespace
cortix scan deep --exclude namespace:kube-system

# Skip a specific deployment (substring match by default)
cortix scan deep --exclude deployment:nginx

# Skip multiple resources
cortix scan deep --exclude namespace:monitoring --exclude deployment:legacy-app

# Exact match only (full name must match)
cortix scan deep --exclude namespace:default -e

# Case-insensitive match
cortix scan deep --exclude deployment:NGINX -i
```

| Flag | Meaning |
|------|---------|
| `--exclude type:name` | Exclude resources where type matches and name contains the pattern |
| `-e` / `--exact` | Require full name equality instead of substring |
| `-i` / `--ignore-case` | Fold case before comparing |
