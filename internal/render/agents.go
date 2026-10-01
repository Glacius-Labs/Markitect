package render

import (
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func renderCodexAgent(r *core.Resource, target string, adapters *core.ProviderAdapters) []byte {
	p := r.Spec.Providers.Codex
	var b strings.Builder
	b.WriteString("# " + Marker + "; source: " + relative(target, r.Path) + "\n")
	if p != nil {
		writeTOMLString(&b, "name", r.Metadata.Name)
		writeTOMLString(&b, "description", r.Spec.Description)
		if p.Model != "" {
			writeTOMLString(&b, "model", p.Model)
		}
		if p.Effort != "" {
			writeTOMLString(&b, "model_reasoning_effort", p.Effort)
		}
		if p.Sandbox != "" {
			writeTOMLString(&b, "sandbox_mode", p.Sandbox)
		}
	}
	instructions := "Read the canonical [agent view](" + relative(target, companionPath(r.Path)) + ") and follow its stated scope."
	if adapters != nil && adapters.InlineAgentText {
		instructions = strings.TrimSpace(r.Spec.Text)
		if adapters.AgentContract != "" {
			instructions += "\n\nShared role-routing and delegation rules: [project agent contract](" + relative(target, adapters.AgentContract) + ")."
		}
		instructions += "\n"
	}
	writeTOMLString(&b, "developer_instructions", instructions)
	return []byte(b.String())
}

func renderClaudeAgent(r *core.Resource, target string, g *core.Graph, adapters *core.ProviderAdapters) []byte {
	p := r.Spec.Providers.Claude
	var b strings.Builder
	b.WriteString("---\nname: " + yamlQuote(r.Metadata.Name) + "\ndescription: " + yamlQuote(r.Spec.Description) + "\n")
	if p != nil {
		if p.Model != "" {
			b.WriteString("model: " + yamlQuote(p.Model) + "\n")
		}
		if p.Effort != "" {
			b.WriteString("effort: " + yamlQuote(p.Effort) + "\n")
		}
		if p.PermissionMode != "" {
			b.WriteString("permissionMode: " + yamlQuote(p.PermissionMode) + "\n")
		}
		if len(p.Tools) > 0 {
			b.WriteString("tools: " + yamlArray(p.Tools) + "\n")
		}
		if len(p.DisallowedTools) > 0 {
			b.WriteString("disallowedTools: " + yamlArray(p.DisallowedTools) + "\n")
		}
		if p.MaxTurns > 0 {
			fmt.Fprintf(&b, "maxTurns: %d\n", p.MaxTurns)
		}
	}
	b.WriteString("---\n\n<!-- " + Marker + "; source: " + relative(target, r.Path) + " -->\n\n")
	if adapters != nil && adapters.InlineAgentText {
		b.WriteString(strings.TrimSpace(r.Spec.Text) + "\n")
		if adapters.AgentContract != "" {
			b.WriteString("\nShared role-routing and delegation rules: [project agent contract](" + relative(target, adapters.AgentContract) + ").\n")
		}
	} else {
		b.WriteString("Read the [canonical agent view](" + relative(target, companionPath(r.Path)) + ") before acting.\n")
	}
	links := dependencyLinks(r, target, g)
	if len(links) > 0 {
		b.WriteString("\n## Dependencies\n\n")
		for _, link := range links {
			writeDependency(&b, link)
		}
	}
	return []byte(b.String())
}
