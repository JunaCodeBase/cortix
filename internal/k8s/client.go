package k8s

import (
	"fmt"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client wraps the typed and dynamic Kubernetes clients.
type Client struct {
	Typed       kubernetes.Interface
	Dynamic     dynamic.Interface
	ClusterName string // kubeconfig context name, or "in-cluster"
}

// NewClient builds a Client following this precedence:
//  1. Explicit --kubeconfig path
//  2. KUBECONFIG environment variable
//  3. ~/.kube/config
//  4. In-cluster service-account token (when running inside a pod)
//
// contextName overrides the active context in the kubeconfig; pass "" to use
// the context that is already selected.
func NewClient(kubeconfigPath, contextName string) (*Client, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		loadingRules.ExplicitPath = kubeconfigPath
	}

	overrides := &clientcmd.ConfigOverrides{}
	if contextName != "" {
		overrides.CurrentContext = contextName
	}

	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)

	cfg, err := cc.ClientConfig()
	if err != nil {
		// Fall back to in-cluster config when running inside a pod.
		inClusterCfg, icErr := rest.InClusterConfig()
		if icErr != nil {
			return nil, fmt.Errorf(
				"k8s: kubeconfig unavailable (%v) and not running in-cluster (%w)",
				err, icErr,
			)
		}
		return buildClient(inClusterCfg, "in-cluster")
	}

	// Derive cluster name from the active kubeconfig context.
	clusterName := "unknown"
	if raw, rawErr := cc.RawConfig(); rawErr == nil && raw.CurrentContext != "" {
		clusterName = raw.CurrentContext
	}
	// A --context override takes precedence as the display name too.
	if contextName != "" {
		clusterName = contextName
	}

	return buildClient(cfg, clusterName)
}

func buildClient(cfg *rest.Config, clusterName string) (*Client, error) {
	typed, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("k8s: typed client: %w", err)
	}

	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("k8s: dynamic client: %w", err)
	}

	return &Client{Typed: typed, Dynamic: dyn, ClusterName: clusterName}, nil
}
