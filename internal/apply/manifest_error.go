package apply

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

// ManifestPreflightError retains a selected manifest's actual language failure
// or unfinished inspection, separately from physical acquisition errors. The
// adapter returns it with known partial inputs when inspection supplies no Root;
// it must not invent a root or capture Definitions before root preflight.
// ModelService preserves the error and observations without calling Compile.
type ManifestPreflightError struct {
	selection  ModelSelection
	inspection model.ManifestResult
}

// NewManifestPreflightError copies an actual InspectManifest result. It neither
// inspects bytes nor establishes acquisition facts. Adapters report I/O,
// redirects and physical boundary failures through ordinary acquisition errors.
func NewManifestPreflightError(selection ModelSelection, inspection model.ManifestResult) *ManifestPreflightError {
	inspection.Diagnostics = cloneDiagnostics(inspection.Diagnostics)
	return &ManifestPreflightError{selection: selection, inspection: inspection}
}

func (e *ManifestPreflightError) Error() string {
	return fmt.Sprintf("manifest preflight %s at %q:%q", e.inspection.Status, e.selection.Boundary, e.selection.ManifestPath)
}

func (e *ManifestPreflightError) Selection() ModelSelection { return e.selection }

func (e *ManifestPreflightError) Inspection() model.ManifestResult {
	result := e.inspection
	result.Diagnostics = cloneDiagnostics(result.Diagnostics)
	return result
}
