package reporter

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/JunaDev/cortixlabs/pkg/types"
)

// PrintJSON encodes the full ScanResult as indented JSON to w.
func PrintJSON(w io.Writer, result *types.ScanResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		return fmt.Errorf("reporter: json encode: %w", err)
	}
	return nil
}
