package projectcli

import (
	"encoding/json"
	"github.com/Glacius-Labs/Markitect/src/internal/application/projectapp"
	"io"
)

func projectOperations() projectapp.Operations { return projectapp.NewOperations() }
func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
