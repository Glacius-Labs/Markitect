package host

import (
	"github.com/Glacius-Labs/Markitect/internal/host/compat/v0_13/consumers/markdown"
	core "github.com/Glacius-Labs/Markitect/internal/host/compat/v0_13/kernel"
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
