package agentrules

import (
	"fmt"
	"path"
	"sort"
	"strings"

	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
)

type outputState struct {
	outputs   map[string][]byte
	owners    map[string]map[string]bool
	canonical map[string]string
}

func generate(model core.SemanticModel, config Config, files map[string][]byte) (map[string][]byte, map[string][]string, error) {
	plan, err := buildPlan(model, config)
	if err != nil {
		return nil, nil, err
	}
	model, config = plan.model, plan.config
	byKey, local := plan.byKey, plan.local
	state := &outputState{outputs: map[string][]byte{}, owners: map[string]map[string]bool{}, canonical: plan.canonical}
	for _, resource := range local {
		if !isMarkitectAIResource(resource) {
			continue
		}
		switch resource.Identity.Kind {
		case "Skill":
			if activeTarget(config, "codex") {
				name := ".agents/skills/" + resource.Identity.Name + "/SKILL.md"
				if err := state.addOwned(name, renderSkill(resource, name, model, config, byKey), resource.Identity.Key); err != nil {
					return nil, nil, err
				}
			}
			if activeTarget(config, "claude") {
				name := ".claude/skills/" + resource.Identity.Name + "/SKILL.md"
				if err := state.addOwned(name, renderSkill(resource, name, model, config, byKey), resource.Identity.Key); err != nil {
					return nil, nil, err
				}
			}
		case "Agent":
			settings := config.AgentSettings[resource.Identity.Key]
			if activeTarget(config, "codex") {
				name := ".codex/agents/" + resource.Identity.Name + ".toml"
				content, err := renderCodexAgent(resource, name, config, settings.Codex, byKey, files)
				if err != nil {
					return nil, nil, err
				}
				if err := state.addOwned(name, content, resource.Identity.Key); err != nil {
					return nil, nil, err
				}
			}
			if activeTarget(config, "claude") {
				name := ".claude/agents/" + resource.Identity.Name + ".md"
				content, err := renderClaudeAgent(resource, name, config, settings.Claude, model, byKey, files)
				if err != nil {
					return nil, nil, err
				}
				if err := state.addOwned(name, content, resource.Identity.Key); err != nil {
					return nil, nil, err
				}
			}
		}
	}

	if activeTarget(config, "claude") {
		if err := renderRuleAdapters(state, config, byKey); err != nil {
			return nil, nil, err
		}
		if err := renderClaudeRuleSources(state, config); err != nil {
			return nil, nil, err
		}
	}
	if hasProviderTarget(config) && config.ProviderAdapters.RoleRegister != "" {
		if err := validRepoPath(config.ProviderAdapters.RoleRegister); err != nil {
			return nil, nil, fmt.Errorf("roleRegister: %w", err)
		}
		if config.ProjectKey != "" && config.ProjectPath != "" {
			for _, provider := range []struct{ target, path string }{
				{"codex", ".agents/roles.md"}, {"claude", ".claude/roles.md"},
			} {
				if !activeTarget(config, provider.target) {
					continue
				}
				body := "<!-- " + Marker + "; source: markitect.yaml -->\n# Repository role assignments\n\nThe canonical person-to-scope assignments are maintained in [the shared role register](" + relative(provider.path, config.ProviderAdapters.RoleRegister) + ").\n"
				if err := state.addOwned(provider.path, []byte(body), config.ProjectKey); err != nil {
					return nil, nil, err
				}
			}
		}
	}
	if err := checkOutputCollisions(state.outputs, state.canonical); err != nil {
		return nil, nil, err
	}
	owners := finishOwners(state.owners)
	if !equalOwners(plan.paths, owners) {
		return nil, nil, fmt.Errorf("generated outputs do not match reserved Agent Rules paths")
	}
	return state.outputs, owners, nil
}

func (state *outputState) addOwned(name string, data []byte, owner string) error {
	if err := validRepoPath(name); err != nil {
		return fmt.Errorf("invalid output path %q: %w", name, err)
	}
	if _, exists := state.outputs[name]; exists {
		return fmt.Errorf("duplicate generated output %q", name)
	}
	state.outputs[name] = data
	if state.owners[name] == nil {
		state.owners[name] = map[string]bool{}
	}
	state.owners[name][owner] = true
	return nil
}

func renderRuleAdapters(state *outputState, config Config, resources map[string]core.ModelResource) error {
	adapters := append([]RuleAdapter(nil), config.RuleAdapters...)
	sort.Slice(adapters, func(i, j int) bool { return adapters[i].Name < adapters[j].Name })
	for _, adapter := range adapters {
		if strings.TrimSpace(adapter.Name) == "" {
			return fmt.Errorf("rule adapter entrypoint name is empty")
		}
		output := ".claude/rules/" + adapter.Name + ".md"
		var b strings.Builder
		b.WriteString("<!-- " + Marker + " -->\n# " + title(adapter.Name) + "\n\n")
		for _, key := range adapter.RuleKeys {
			resource, ok := resources[key]
			if !ok {
				return fmt.Errorf("rule adapter %q references unresolved Rule %q", adapter.Name, key)
			}
			if resource.Identity.Kind != "Rule" {
				return fmt.Errorf("rule adapter %q source %s must be Rule", adapter.Name, key)
			}
			if resource.Identity.Package != "" {
				return fmt.Errorf("external ruleAdapters are not supported: %q", key)
			}
			b.WriteString("- [" + resource.Identity.Name + "](" + relative(output, resource.Source.Path) + ")\n")
			if state.owners[output] == nil {
				state.owners[output] = map[string]bool{}
			}
			state.owners[output][key] = true
		}
		if err := state.addOwned(output, []byte(b.String()), config.ProjectKey); err != nil {
			return err
		}
		delete(state.owners[output], config.ProjectKey)
		if len(state.owners[output]) == 0 {
			delete(state.owners, output)
		}
	}
	return nil
}

func renderClaudeRuleSources(state *outputState, config Config) error {
	adapters := config.ProviderAdapters
	names := make([]string, 0, len(adapters.ClaudeRuleSources))
	for name := range adapters.ClaudeRuleSources {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		sources := adapters.ClaudeRuleSources[name]
		output := ".claude/rules/" + name + ".md"
		var b strings.Builder
		b.WriteString("<!-- " + Marker + "; source: markitect.yaml -->\n# " + adapterTitle(name) + "\n\nRead the linked canonical guidance before acting; it owns the operative requirements.\n\n")
		for _, source := range sources {
			if err := validRepoPath(source); err != nil {
				return fmt.Errorf("ruleSources.%s: %w", name, err)
			}
			label := strings.TrimSuffix(path.Base(source), path.Ext(source))
			b.WriteString("- [" + adapterTitle(label) + "](" + relative(output, source) + ")\n")
		}
		if err := state.addOwned(output, []byte(b.String()), config.ProjectKey); err != nil {
			return err
		}
	}
	return nil
}

func checkOutputCollisions(outputs map[string][]byte, canonical map[string]string) error {
	seen := map[string]string{}
	for source := range canonical {
		seen[source] = "source " + canonical[source]
	}
	paths := make([]string, 0, len(outputs))
	for name := range outputs {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	for _, name := range paths {
		key := foldPath(name)
		if prior, exists := seen[key]; exists {
			return fmt.Errorf("output path collision: %s conflicts with %q", prior, name)
		}
		seen[key] = "output " + name
	}
	return nil
}

func finishOwners(values map[string]map[string]bool) map[string][]string {
	owners := make(map[string][]string, len(values))
	for path, set := range values {
		for key := range set {
			owners[path] = append(owners[path], key)
		}
		sort.Strings(owners[path])
	}
	return owners
}

func sortDiagnostics(values []core.Diagnostic) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].Path != values[j].Path {
			return values[i].Path < values[j].Path
		}
		if values[i].Code != values[j].Code {
			return values[i].Code < values[j].Code
		}
		if values[i].Line != values[j].Line {
			return values[i].Line < values[j].Line
		}
		return values[i].Message < values[j].Message
	})
}

func foldPath(value string) string { return strings.ToLower(value) }

func validRepoPath(value string) error {
	if value == "" || strings.ContainsAny(value, "\\:") || strings.ContainsRune(value, 0) || strings.HasPrefix(value, "/") || path.Clean(value) != value || value == "." || strings.HasPrefix(value, "../") {
		return fmt.Errorf("path must be repository-relative and normalized")
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("path contains unsafe component")
		}
	}
	return nil
}

func title(value string) string { return strings.ReplaceAll(value, "-", " ") }

func adapterTitle(value string) string {
	words := strings.Fields(strings.ReplaceAll(value, "-", " "))
	for i, word := range words {
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}
