package host

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/inputs"
	"github.com/Glacius-Labs/Markitect/internal/modules/agentrules"
	"github.com/Glacius-Labs/Markitect/internal/modules/markdown"
)

// GenerateOutputs composes independent projections from one normalized model.
// Host owns target selection and collision enforcement; Modules never consume
// another Module's output as semantic input. All bytes come from p.Snapshot.
func GenerateOutputs(p *Project) (map[string][]byte, error) {
	outputs, _, err := GenerateOutputsWithOwners(p)
	return outputs, err
}

func GenerateOutputsWithOwners(p *Project) (map[string][]byte, map[string][]string, error) {
	model, err := CompileModel(p)
	if err != nil {
		return nil, nil, err
	}
	documents, documentOwners, err := markdown.Generate(model, markdownConfig(p), p.Snapshot.Files)
	if err != nil {
		return nil, nil, err
	}
	agents, agentOwners, err := agentrules.Generate(model, agentRulesConfig(p), p.Snapshot.Files)
	if err != nil {
		return nil, nil, err
	}
	folded := map[string]string{}
	for _, output := range []map[string][]byte{documents, agents} {
		for _, name := range sortedFiles(output) {
			key := strings.ToLower(name)
			if prior, exists := folded[key]; exists {
				return nil, nil, fmt.Errorf("generated output collision: %q and %q", prior, name)
			}
			folded[key] = name
		}
	}
	for name, data := range agents {
		documents[name] = data
		documentOwners[name] = agentOwners[name]
	}
	return documents, documentOwners, nil
}

func markdownConfig(p *Project) markdown.Config {
	config := markdown.Config{Packages: map[string]string{}}
	if p == nil || p.Graph == nil || p.Graph.Project == nil {
		return config
	}
	project := p.Graph.Project
	config.ProjectKey, config.Path = project.GraphKey(), project.Path
	for _, target := range project.Spec.Targets {
		if strings.EqualFold(target, "markdown") {
			config.Enabled = true
		}
	}
	for _, area := range project.Spec.Areas {
		config.Areas = append(config.Areas, markdown.Area{Name: area.Name, Path: area.Path})
	}
	for name, manifest := range p.Graph.Packages {
		config.Packages[name] = manifest.Spec.Version
	}
	if project.Spec.Consistency != nil {
		config.FunctionalPredicates = append([]string(nil), project.Spec.Consistency.FunctionalPredicates...)
	}
	if project.Spec.Documentation != nil {
		config.DocumentationRoots = append([]string(nil), project.Spec.Documentation.Roots...)
	}
	return config
}

// agentRulesConfig lowers the supported source configuration at the Host
// boundary. Provider-specific types belong to Agent Rules, not Core.
func agentRulesConfig(p *Project) agentrules.Config {
	config := agentrules.Config{AgentSettings: map[string]agentrules.AgentSettings{}, PackageVersions: map[string]string{}}
	if p == nil || p.Graph == nil || p.Graph.Project == nil {
		return config
	}
	project := p.Graph.Project
	config.ProjectKey, config.ProjectPath = project.GraphKey(), project.Path
	for _, target := range project.Spec.Targets {
		config.Targets = append(config.Targets, strings.ToLower(target))
	}
	names := make([]string, 0, len(project.Spec.RuleAdapters))
	for name := range project.Spec.RuleAdapters {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		mapping := agentrules.RuleAdapter{Name: name}
		for _, ref := range project.Spec.RuleAdapters[name] {
			mapping.RuleKeys = append(mapping.RuleKeys, ref.GraphKey("", "", "Rule"))
		}
		config.RuleAdapters = append(config.RuleAdapters, mapping)
	}
	if a := project.Spec.ProviderAdapters; a != nil {
		config.ProviderAdapters = agentrules.ProviderAdapters{AgentContract: a.AgentContract, RoleRegister: a.RoleRegister, InlineAgentText: a.InlineAgentText, StrictInventory: a.StrictInventory, ClaudeRuleSources: a.RuleSources, RetiredSkills: append([]string(nil), a.RetiredSkills...), RetiredAgents: append([]string(nil), a.RetiredAgents...)}
	}
	for name, manifest := range p.Graph.Packages {
		config.PackageVersions[name] = manifest.Spec.Version
	}
	for key, resource := range p.Graph.Resources {
		if resource.APIVersion != core.APIVersion || resource.Kind != "Agent" {
			continue
		}
		settings := agentrules.AgentSettings{}
		if c := resource.Spec.Providers.Codex; c != nil {
			settings.Codex = &agentrules.CodexSettings{Model: c.Model, Effort: c.Effort, Sandbox: c.Sandbox}
		}
		if c := resource.Spec.Providers.Claude; c != nil {
			settings.Claude = &agentrules.ClaudeSettings{Model: c.Model, Effort: c.Effort, PermissionMode: c.PermissionMode, Tools: append([]string(nil), c.Tools...), DisallowedTools: append([]string(nil), c.DisallowedTools...), MaxTurns: c.MaxTurns}
		}
		config.AgentSettings[key] = settings
	}
	return config
}

func checkAgentRules(p *Project) []core.Diagnostic {
	model, err := CompileModel(p)
	if err != nil {
		return []core.Diagnostic{{Code: "agent-rules.model", Message: err.Error()}}
	}
	return agentrules.Validate(model, agentRulesConfig(p), p.Snapshot.Files)
}

// projectionScope asks each concrete capability for path ownership without
// reading prose or running module evidence checks during source validation.
func projectionScope(p *Project) (inputs.ProjectionScope, error) {
	model, err := CompileModel(p)
	if err != nil {
		return inputs.ProjectionScope{}, err
	}
	views, err := markdown.ViewPaths(model, markdownConfig(p))
	if err != nil {
		return inputs.ProjectionScope{}, err
	}
	documents, err := markdown.OutputPaths(model, markdownConfig(p))
	if err != nil {
		return inputs.ProjectionScope{}, err
	}
	agents, err := agentrules.OutputPaths(model, agentRulesConfig(p))
	if err != nil {
		return inputs.ProjectionScope{}, err
	}
	scope := inputs.ProjectionScope{TypedViews: map[string]string{}, OutputPaths: map[string]bool{}}
	for name, r := range views {
		scope.TypedViews[name] = r.Identity.Key
	}
	folded := map[string]string{}
	for _, paths := range []map[string][]string{documents, agents} {
		for _, name := range sortedResourcePaths(paths) {
			key := strings.ToLower(name)
			if prior, ok := folded[key]; ok {
				return inputs.ProjectionScope{}, fmt.Errorf("generated output collision: %q and %q", prior, name)
			}
			folded[key] = name
			scope.OutputPaths[name] = true
		}
	}
	return scope, nil
}
func sortedResourcePaths(paths map[string][]string) []string {
	names := make([]string, 0, len(paths))
	for name := range paths {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
