package agentrules

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

type projectionPlan struct {
	model     core.SemanticModel
	config    Config
	resources []core.ModelResource
	local     []core.ModelResource
	byKey     map[string]core.ModelResource
	canonical map[string]string
	paths     map[string][]string
}

func buildPlan(model core.SemanticModel, config Config) (*projectionPlan, error) {
	plan := &projectionPlan{
		model:     model,
		config:    config,
		resources: append([]core.ModelResource(nil), model.Resources...),
		byKey:     make(map[string]core.ModelResource, len(model.Resources)),
		canonical: map[string]string{},
		paths:     map[string][]string{},
	}
	sort.Slice(plan.resources, func(i, j int) bool {
		if plan.resources[i].Source.Path != plan.resources[j].Source.Path {
			return plan.resources[i].Source.Path < plan.resources[j].Source.Path
		}
		return plan.resources[i].Identity.Key < plan.resources[j].Identity.Key
	})
	for _, resource := range plan.resources {
		if resource.Identity.Key == "" {
			return nil, fmt.Errorf("normalized resource at %q has no identity key", resource.Source.Path)
		}
		if _, exists := plan.byKey[resource.Identity.Key]; exists {
			return nil, fmt.Errorf("duplicate normalized resource key %q", resource.Identity.Key)
		}
		plan.byKey[resource.Identity.Key] = resource
		if resource.Identity.Package != "" {
			continue
		}
		if err := addCanonicalPath(plan.canonical, resource.Source.Path, "source", false); err != nil {
			return nil, fmt.Errorf("resource %s source path: %w", resource.Identity.Key, err)
		}
		plan.local = append(plan.local, resource)
	}
	if config.ProjectPath != "" {
		if err := addCanonicalPath(plan.canonical, config.ProjectPath, "Project source", true); err != nil {
			return nil, err
		}
	}

	for _, target := range config.Targets {
		if target != "codex" && target != "claude" && target != "markdown" {
			return nil, fmt.Errorf("unsupported provider target %q", target)
		}
	}

	for _, resource := range plan.local {
		if !isMarkitectAIResource(resource) {
			continue
		}
		switch resource.Identity.Kind {
		case "Skill":
			if activeTarget(config, "codex") {
				if err := plan.add(".agents/skills/"+resource.Identity.Name+"/SKILL.md", resource.Identity.Key); err != nil {
					return nil, err
				}
			}
			if activeTarget(config, "claude") {
				if err := plan.add(".claude/skills/"+resource.Identity.Name+"/SKILL.md", resource.Identity.Key); err != nil {
					return nil, err
				}
			}
		case "Agent":
			if activeTarget(config, "codex") {
				if err := plan.add(".codex/agents/"+resource.Identity.Name+".toml", resource.Identity.Key); err != nil {
					return nil, err
				}
			}
			if activeTarget(config, "claude") {
				if err := plan.add(".claude/agents/"+resource.Identity.Name+".md", resource.Identity.Key); err != nil {
					return nil, err
				}
			}
		}
	}

	if activeTarget(config, "claude") {
		adapters := append([]RuleAdapter(nil), config.RuleAdapters...)
		sort.Slice(adapters, func(i, j int) bool { return adapters[i].Name < adapters[j].Name })
		for _, adapter := range adapters {
			if strings.TrimSpace(adapter.Name) == "" {
				return nil, fmt.Errorf("rule adapter entrypoint name is empty")
			}
			output := ".claude/rules/" + adapter.Name + ".md"
			for _, key := range adapter.RuleKeys {
				resource, ok := plan.byKey[key]
				if !ok {
					return nil, fmt.Errorf("rule adapter %q references unresolved Rule %q", adapter.Name, key)
				}
				if resource.Identity.Kind != "Rule" || !isMarkitectAIResource(resource) {
					return nil, fmt.Errorf("rule adapter %q source %s must be a Markitect AI Rule", adapter.Name, key)
				}
				if resource.Identity.Package != "" {
					return nil, fmt.Errorf("external ruleAdapters are not supported: %q", key)
				}
			}
			if err := plan.add(output, adapter.RuleKeys...); err != nil {
				return nil, err
			}
		}
		names := make([]string, 0, len(config.ProviderAdapters.ClaudeRuleSources))
		for name := range config.ProviderAdapters.ClaudeRuleSources {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if !providerName(name) || len(config.ProviderAdapters.ClaudeRuleSources[name]) == 0 {
				return nil, fmt.Errorf("rule source adapter %q needs a valid name and at least one source", name)
			}
			if config.ProjectKey == "" || config.ProjectPath == "" {
				return nil, fmt.Errorf("Project identity and source path are required for provider entrypoints")
			}
			if err := plan.add(".claude/rules/"+name+".md", config.ProjectKey); err != nil {
				return nil, err
			}
		}
	}
	if hasProviderTarget(config) && config.ProviderAdapters.RoleRegister != "" {
		if config.ProjectKey == "" || config.ProjectPath == "" {
			return nil, fmt.Errorf("Project identity and source path are required for provider entrypoints")
		}
		if activeTarget(config, "codex") {
			if err := plan.add(".agents/roles.md", config.ProjectKey); err != nil {
				return nil, err
			}
		}
		if activeTarget(config, "claude") {
			if err := plan.add(".claude/roles.md", config.ProjectKey); err != nil {
				return nil, err
			}
		}
	}
	seen := make(map[string]string, len(plan.canonical)+len(plan.paths))
	for path, source := range plan.canonical {
		seen[path] = "source " + source
	}
	for output := range plan.paths {
		folded := foldPath(output)
		if prior, exists := seen[folded]; exists {
			return nil, fmt.Errorf("output path collision: %s conflicts with %q", prior, output)
		}
		seen[folded] = "output " + output
	}
	for _, owners := range plan.paths {
		sort.Strings(owners)
	}
	return plan, nil
}

func (plan *projectionPlan) add(path string, owners ...string) error {
	if err := validRepoPath(path); err != nil {
		return fmt.Errorf("invalid output path %q: %w", path, err)
	}
	if _, exists := plan.paths[path]; exists {
		return fmt.Errorf("duplicate generated output %q", path)
	}
	seen := map[string]bool{}
	for _, owner := range owners {
		if owner == "" || seen[owner] {
			continue
		}
		seen[owner] = true
		plan.paths[path] = append(plan.paths[path], owner)
	}
	return nil
}

func addCanonicalPath(paths map[string]string, value, label string, allowExact bool) error {
	if err := validRepoPath(value); err != nil {
		return fmt.Errorf("%s path: %w", label, err)
	}
	folded := foldPath(value)
	if prior, exists := paths[folded]; exists {
		if !allowExact || prior != value {
			return fmt.Errorf("source path collision: %q and %q", prior, value)
		}
		return nil
	}
	paths[folded] = value
	return nil
}

func equalOwners(planned, rendered map[string][]string) bool {
	if len(planned) != len(rendered) {
		return false
	}
	for path, owners := range planned {
		other, exists := rendered[path]
		if !exists || len(owners) != len(other) {
			return false
		}
		for index := range owners {
			if owners[index] != other[index] {
				return false
			}
		}
	}
	return true
}
