package export

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// Writer handles writing an ExportResult to disk.
type Writer struct {
	outputDir string
}

// NewWriter creates a Writer that will write to outputDir.
func NewWriter(outputDir string) *Writer {
	return &Writer{outputDir: outputDir}
}

// Write creates the full directory structure and writes all files.
func (w *Writer) Write(result *ExportResult) error {
	if err := os.MkdirAll(w.outputDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	// README
	readme := fmt.Sprintf(readmeTemplate, result.ClusterName, time.Now().Format("2006-01-02 15:04:05 UTC"))
	if err := w.writeFile("README.md", []byte(readme)); err != nil {
		return err
	}

	// kustomization.yaml
	if err := w.writeFile("kustomization.yaml", GenerateKustomization(result)); err != nil {
		return err
	}

	// apply.sh
	script := GenerateApplyScript(result)
	path := filepath.Join(w.outputDir, "apply.sh")
	if err := os.WriteFile(path, script, 0o755); err != nil {
		return fmt.Errorf("write apply.sh: %w", err)
	}

	// WARNINGS.md
	if warnings := GenerateWarnings(result); warnings != nil {
		if err := w.writeFile("WARNINGS.md", warnings); err != nil {
			return err
		}
	}

	// HELM_RELEASES.md
	if doc := HelmReleasesDoc(result); doc != nil {
		if err := w.writeFile("HELM_RELEASES.md", doc); err != nil {
			return err
		}
	}

	// Cluster-wide resources
	if err := w.writeCluster(&result.Cluster); err != nil {
		return err
	}

	// Namespaced resources
	for _, ns := range result.Namespaces {
		if err := w.writeNamespace(&ns); err != nil {
			return err
		}
	}

	return nil
}

func (w *Writer) writeCluster(ce *ClusterExport) error {
	dirs := map[string][]ExportedResource{
		"cluster/namespaces":          ce.Namespaces,
		"cluster/storageclasses":      ce.StorageClasses,
		"cluster/clusterroles":        ce.ClusterRoles,
		"cluster/clusterrolebindings": ce.ClusterRoleBindings,
	}
	for dir, resources := range dirs {
		if err := w.writeResources(dir, resources); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) writeNamespace(ns *NamespaceExport) error {
	base := filepath.Join("namespaces", ns.Name)
	dirs := map[string][]ExportedResource{
		filepath.Join(base, "deployments"):     ns.Deployments,
		filepath.Join(base, "statefulsets"):    ns.StatefulSets,
		filepath.Join(base, "services"):        ns.Services,
		filepath.Join(base, "ingresses"):       ns.Ingresses,
		filepath.Join(base, "configmaps"):      ns.ConfigMaps,
		filepath.Join(base, "secrets"):         ns.Secrets,
		filepath.Join(base, "serviceaccounts"): ns.ServiceAccounts,
		filepath.Join(base, "hpas"):            ns.HPAs,
	}
	for dir, resources := range dirs {
		if err := w.writeResources(dir, resources); err != nil {
			return err
		}
	}

	// Helm values stubs
	for _, hr := range ns.HelmReleases {
		helmDir := filepath.Join(base, "helm")
		stub := HelmValuesStub(hr)
		if err := w.writeResourceInDir(helmDir, hr.Name+"-values.yaml", stub); err != nil {
			return err
		}
	}

	return nil
}

func (w *Writer) writeResources(relDir string, resources []ExportedResource) error {
	if len(resources) == 0 {
		return nil
	}
	absDir := filepath.Join(w.outputDir, relDir)
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", absDir, err)
	}
	for _, r := range resources {
		path := filepath.Join(absDir, r.Filename)
		if err := os.WriteFile(path, r.Content, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		slog.Debug("wrote resource", "path", path)
	}
	return nil
}

func (w *Writer) writeResourceInDir(relDir, filename string, content []byte) error {
	absDir := filepath.Join(w.outputDir, relDir)
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", absDir, err)
	}
	path := filepath.Join(absDir, filename)
	return os.WriteFile(path, content, 0o644)
}

func (w *Writer) writeFile(relPath string, content []byte) error {
	path := filepath.Join(w.outputDir, relPath)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", relPath, err)
	}
	return nil
}
