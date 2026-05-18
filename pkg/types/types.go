package types

import "time"

// ToolStatus represents whether a tool was detected in the cluster.
type ToolStatus string

const (
	ToolFound   ToolStatus = "found"
	ToolMissing ToolStatus = "missing"
)

// DetectedTool holds the result for a single tool check.
type DetectedTool struct {
	Name      string     `json:"name"`
	Namespace string     `json:"namespace,omitempty"`
	Status    ToolStatus `json:"status"`
	Version   string     `json:"version,omitempty"`
}

// DetectionResult is the top-level output of a scan.
type DetectionResult struct {
	ClusterName string         `json:"cluster_name"`
	ScannedAt   time.Time      `json:"scanned_at"`
	Found       []DetectedTool `json:"found"`
	Missing     []DetectedTool `json:"missing"`
}
