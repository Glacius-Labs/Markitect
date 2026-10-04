package host

import (
	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/modules/markdown"
)

// CheckConsistency supplies the normalized model and explicit literal-assertion
// configuration to the Markdown capability. It never interprets prose itself.
func CheckConsistency(p *Project) []core.Diagnostic {
	if p == nil || p.Graph == nil || p.Graph.Project == nil || p.Graph.Project.Spec.Consistency == nil {
		return nil
	}
	model, err := CompileModel(p)
	if err != nil {
		return []core.Diagnostic{{Code: "markdown.model", Message: err.Error()}}
	}
	config := markdownConfig(p)
	config.DocumentationRoots = nil
	return markdown.Validate(model, config, p.Snapshot.Files)
}
