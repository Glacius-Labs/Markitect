package app

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func checkProviderAdapterInputs(p *Project) []core.Diagnostic {
	if p.Graph.Project == nil || p.Graph.Project.Spec.ProviderAdapters == nil {
		return nil
	}
	a := p.Graph.Project.Spec.ProviderAdapters
	targets := map[string]bool{}
	for _, target := range p.Graph.Project.Spec.Targets {
		targets[strings.ToLower(target)] = true
	}
	var paths []string
	if targets["claude"] {
		for _, sources := range a.RuleSources {
			paths = append(paths, sources...)
		}
	}
	if a.AgentContract != "" && (targets["codex"] || targets["claude"]) {
		paths = append(paths, a.AgentContract)
	}
	if a.RoleRegister != "" && (targets["codex"] || targets["claude"]) {
		paths = append(paths, a.RoleRegister)
	}
	sort.Strings(paths)
	var findings []core.Diagnostic
	for _, name := range paths {
		if !safeAssertionPath(name) {
			findings = append(findings, core.Diagnostic{Code: "provider-adapter.path", Path: "markitect.yaml", Message: fmt.Sprintf("invalid adapter source path %q", name)})
			continue
		}
		if _, ok := p.Snapshot.Files[name]; !ok {
			findings = append(findings, core.Diagnostic{Code: "provider-adapter.source", Path: name, Message: "adapter source is missing"})
		}
	}
	if a.StrictInventory {
		mapped := map[string]bool{}
		if targets["claude"] {
			for _, sources := range a.RuleSources {
				for _, name := range sources {
					mapped[name] = true
				}
			}
		}
		keys := make([]string, 0, len(p.Graph.Resources))
		for key := range p.Graph.Resources {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			resource := p.Graph.Resources[key]
			if resource.Package != "" {
				continue
			}
			if resource.Kind == "Rule" && targets["claude"] {
				view := strings.TrimSuffix(resource.Path, filepath.Ext(resource.Path)) + ".md"
				if !mapped[view] {
					findings = append(findings, core.Diagnostic{Code: "provider-adapter.unmapped-rule", Path: resource.Path, Line: resource.Line, Message: "canonical Rule has no provider rule adapter mapping"})
				}
			}
			if resource.Kind == "Agent" {
				codex, claude := resource.Spec.Providers.Codex, resource.Spec.Providers.Claude
				if (targets["codex"] && (codex == nil || codex.Model == "" || codex.Effort == "" || codex.Sandbox == "")) || (targets["claude"] && (claude == nil || claude.Model == "" || claude.Effort == "" || claude.PermissionMode == "")) {
					findings = append(findings, core.Diagnostic{Code: "provider-adapter.settings", Path: resource.Path, Line: resource.Line, Message: "strict provider Agent requires explicit model, effort and safety settings for each selected target"})
				}
			}
		}
	}
	return findings
}

func checkProviderInventory(p *Project, outputs map[string][]byte) []core.Diagnostic {
	if p.Graph.Project == nil || p.Graph.Project.Spec.ProviderAdapters == nil || !p.Graph.Project.Spec.ProviderAdapters.StrictInventory {
		return nil
	}
	a := p.Graph.Project.Spec.ProviderAdapters
	retired := map[string]bool{}
	for _, name := range a.RetiredSkills {
		for _, prefix := range []string{".agents/skills/", ".claude/skills/"} {
			retired[prefix+name+"/SKILL.md"] = true
		}
	}
	for _, name := range a.RetiredAgents {
		retired[".codex/agents/"+name+".toml"] = true
		retired[".claude/agents/"+name+".md"] = true
	}
	var findings []core.Diagnostic
	for _, name := range sortedFiles(p.Snapshot.Files) {
		managed := strings.HasPrefix(name, ".agents/rules/") || strings.HasPrefix(name, ".claude/rules/") || strings.HasPrefix(name, ".agents/skills/") || strings.HasPrefix(name, ".claude/skills/") || strings.HasPrefix(name, ".codex/agents/") || strings.HasPrefix(name, ".claude/agents/") || name == ".agents/roles.md" || name == ".claude/roles.md"
		if !managed {
			continue
		}
		if retired[name] {
			findings = append(findings, core.Diagnostic{Code: "retired-output", Path: name, Message: "retired provider adapter still exists"})
			continue
		}
		if _, ok := outputs[name]; !ok {
			findings = append(findings, core.Diagnostic{Code: "unregistered-output", Path: name, Message: "provider adapter is not declared by canonical resources"})
		}
	}
	return findings
}
