package host

import (
	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/modules/markdown"
)

// CheckDocumentationRouters delegates literal snapshot link checks. Links are
// documentation navigation and do not create semantic graph dependencies.
func CheckDocumentationRouters(p *Project) []core.Diagnostic {
	if p == nil || p.Graph == nil || p.Graph.Project == nil || p.Graph.Project.Spec.Documentation == nil {
		return nil
	}
	return markdown.ValidateDocumentationRouters(p.Graph.Project.Spec.Documentation.Roots, p.Snapshot.Files)
}
