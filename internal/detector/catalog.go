package detector

// All standard observability tool detectors.
// Each uses app.kubernetes.io/name — the canonical label set by all major Helm charts.
var (
	Grafana       = NewPodLabel("Grafana",        "app.kubernetes.io/name=grafana")
	AlertManager  = NewPodLabel("AlertManager",   "app.kubernetes.io/name=alertmanager")
	Loki          = NewPodLabel("Loki",           "app.kubernetes.io/name=loki")
	MetricsServer = NewPodLabel("metrics-server", "app.kubernetes.io/name=metrics-server")
	CertManager   = NewPodLabel("cert-manager",   "app.kubernetes.io/name=cert-manager")
	IngressNginx  = NewPodLabel("ingress-nginx",  "app.kubernetes.io/name=ingress-nginx")
)
