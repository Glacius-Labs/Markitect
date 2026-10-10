package agentrules

import (
	"fmt"
	"sort"
	"strings"

	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
)

func validateNormalizedConfig(config Config, resources []core.ModelResource, byKey map[string]core.ModelResource) []core.Diagnostic {
	var findings []core.Diagnostic
	seenNames := map[string]map[string]core.ModelResource{}
	for _, resource := range resources {
		kind := resource.Identity.Kind
		if resource.Identity.Package != "" || !isMarkitectAIResource(resource) || kind != "Skill" && kind != "Agent" {
			continue
		}
		if seenNames[kind] == nil {
			seenNames[kind] = map[string]core.ModelResource{}
		}
		if prior, exists := seenNames[kind][resource.Identity.Name]; exists {
			findings = append(findings, diagnostic("provider.name", resource.Source.Path, resource.Source.Line, fmt.Sprintf("provider name %q is duplicated for kind %s (also %s)", resource.Identity.Name, kind, prior.Identity.Key)))
		} else {
			seenNames[kind][resource.Identity.Name] = resource
		}
	}

	ruleNames := map[string]bool{}
	for _, adapter := range config.RuleAdapters {
		if strings.TrimSpace(adapter.Name) == "" {
			findings = append(findings, diagnostic("rule-adapter.name", config.ProjectPath, 0, "rule adapter entrypoint name is empty"))
		}
		if ruleNames[adapter.Name] {
			findings = append(findings, diagnostic("rule-adapter.name", config.ProjectPath, 0, fmt.Sprintf("rule adapter entrypoint name %q is duplicated", adapter.Name)))
		}
		ruleNames[adapter.Name] = true
		for _, key := range adapter.RuleKeys {
			resource, ok := byKey[key]
			if !ok {
				findings = append(findings, diagnostic("rule-adapter.reference", config.ProjectPath, 0, fmt.Sprintf("rule adapter %q references unresolved Rule %q", adapter.Name, key)))
				continue
			}
			if resource.Identity.Kind != "Rule" {
				findings = append(findings, diagnostic("rule-adapter.kind", resource.Source.Path, resource.Source.Line, fmt.Sprintf("rule adapter source %s must be Rule", key)))
			} else if !isMarkitectAIResource(resource) {
				findings = append(findings, diagnostic("rule-adapter.api", resource.Source.Path, resource.Source.Line, fmt.Sprintf("rule adapter source %s must use Markitect AI API %s", key, MarkitectAIVersion)))
			}
			if resource.Identity.Package != "" {
				findings = append(findings, diagnostic("rule-adapter.package", config.ProjectPath, 0, "ruleAdapters cannot directly reference package resources; use an explicit local wrapper"))
			}
		}
	}

	provider := config.ProviderAdapters
	if (provider.RoleRegister != "" && hasProviderTarget(config) || activeTarget(config, "claude") && len(provider.ClaudeRuleSources) != 0) && (config.ProjectKey == "" || config.ProjectPath == "") {
		findings = append(findings, diagnostic("provider-adapter.project", config.ProjectPath, 0, "Project identity and source path are required for provider entrypoints"))
	}
	for name, sources := range provider.ClaudeRuleSources {
		if !providerName(name) || len(sources) == 0 {
			findings = append(findings, diagnostic("provider-adapter.rule", config.ProjectPath, 0, fmt.Sprintf("rule adapter %q needs a valid name and at least one source", name)))
		}
	}
	for _, item := range []struct {
		kind  string
		names []string
	}{{"Skill", provider.RetiredSkills}, {"Agent", provider.RetiredAgents}} {
		seen := map[string]bool{}
		for _, name := range item.names {
			if !providerName(name) {
				findings = append(findings, diagnostic("provider-adapter.retired-name", config.ProjectPath, 0, fmt.Sprintf("retired %s name %q is invalid", item.kind, name)))
			}
			if seen[name] {
				findings = append(findings, diagnostic("provider-adapter.retired-duplicate", config.ProjectPath, 0, fmt.Sprintf("retired %s name %q is duplicated", item.kind, name)))
			}
			seen[name] = true
			for _, resource := range resources {
				if resource.Identity.Package == "" && isMarkitectAIResource(resource) && resource.Identity.Kind == item.kind && resource.Identity.Name == name {
					findings = append(findings, diagnostic("provider-adapter.retired-active", config.ProjectPath, 0, fmt.Sprintf("retired %s name %q is still an active resource", item.kind, name)))
				}
			}
		}
	}

	if provider.StrictInventory {
		mappedRules := map[string]bool{}
		if activeTarget(config, "claude") {
			for _, sources := range provider.ClaudeRuleSources {
				for _, source := range sources {
					mappedRules[source] = true
				}
			}
			for _, adapter := range config.RuleAdapters {
				for _, key := range adapter.RuleKeys {
					if rule, ok := byKey[key]; ok && isMarkitectAIResource(rule) && rule.Identity.Kind == "Rule" && rule.Identity.Package == "" {
						mappedRules[rule.Source.Path] = true
					}
				}
			}
		}
		for _, resource := range resources {
			if resource.Identity.Package != "" || !isMarkitectAIResource(resource) {
				continue
			}
			if resource.Identity.Kind == "Rule" && activeTarget(config, "claude") && !mappedRules[resource.Source.Path] {
				findings = append(findings, diagnostic("provider-adapter.unmapped-rule", resource.Source.Path, resource.Source.Line, "canonical Rule has no provider rule adapter mapping"))
			}
			if resource.Identity.Kind == "Agent" {
				settings := config.AgentSettings[resource.Identity.Key]
				codex, claude := settings.Codex, settings.Claude
				codexMissing := activeTarget(config, "codex") && (codex == nil || codex.Model == "" || codex.Effort == "" || codex.Sandbox == "")
				claudeMissing := activeTarget(config, "claude") && (claude == nil || claude.Model == "" || claude.Effort == "" || claude.PermissionMode == "")
				if codexMissing || claudeMissing {
					findings = append(findings, diagnostic("provider-adapter.settings", resource.Source.Path, resource.Source.Line, "strict provider Agent requires explicit model, effort and safety settings for each selected target"))
				}
			}
		}
	}
	sortDiagnostics(findings)
	return findings
}

func validateAdapters(config Config, files map[string][]byte, outputs map[string][]string) []core.Diagnostic {
	provider := config.ProviderAdapters
	targets := activeTarget(config, "codex") || activeTarget(config, "claude")
	var paths []string
	if activeTarget(config, "claude") {
		for _, sources := range provider.ClaudeRuleSources {
			paths = append(paths, sources...)
		}
	}
	if targets && provider.AgentContract != "" {
		paths = append(paths, provider.AgentContract)
	}
	if targets && provider.RoleRegister != "" {
		paths = append(paths, provider.RoleRegister)
	}
	sort.Strings(paths)
	generated := make(map[string]bool, len(outputs))
	for name := range outputs {
		generated[strings.ToLower(name)] = true
	}
	var findings []core.Diagnostic
	for _, name := range paths {
		if !safeInputPath(name) {
			findings = append(findings, diagnostic("provider-adapter.path", config.ProjectPath, 0, fmt.Sprintf("invalid adapter source path %q", name)))
			continue
		}
		if generated[strings.ToLower(name)] {
			findings = append(findings, diagnostic("provider-adapter.generated-source", name, 0, "provider adapter sources must not point to generated Markitect outputs"))
			continue
		}
		data, ok := files[name]
		if !ok {
			findings = append(findings, diagnostic("provider-adapter.source", name, 0, "adapter source is missing"))
		} else if IsGenerated(data) {
			findings = append(findings, diagnostic("provider-adapter.generated-source", name, 0, "provider adapter sources must not point to generated Markitect outputs"))
		}
	}
	if provider.StrictInventory {
		findings = append(findings, validateProviderInventory(config, files, outputs)...)
	}
	sortDiagnostics(findings)
	return findings
}

func validateProviderInventory(config Config, files map[string][]byte, outputs map[string][]string) []core.Diagnostic {
	provider := config.ProviderAdapters
	retired := map[string]bool{}
	for _, name := range provider.RetiredSkills {
		retired[".agents/skills/"+name+"/SKILL.md"] = true
		retired[".claude/skills/"+name+"/SKILL.md"] = true
	}
	for _, name := range provider.RetiredAgents {
		retired[".codex/agents/"+name+".toml"] = true
		retired[".claude/agents/"+name+".md"] = true
	}
	var findings []core.Diagnostic
	paths := make([]string, 0, len(files))
	for name := range files {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	for _, name := range paths {
		managed := strings.HasPrefix(name, ".agents/rules/") || strings.HasPrefix(name, ".claude/rules/") || strings.HasPrefix(name, ".agents/skills/") || strings.HasPrefix(name, ".claude/skills/") || strings.HasPrefix(name, ".codex/agents/") || strings.HasPrefix(name, ".claude/agents/") || name == ".agents/roles.md" || name == ".claude/roles.md"
		if !managed {
			continue
		}
		if retired[name] {
			findings = append(findings, diagnostic("retired-output", name, 0, "retired provider adapter still exists"))
			continue
		}
		if _, ok := outputs[name]; !ok {
			findings = append(findings, diagnostic("unregistered-output", name, 0, "provider adapter is not declared by canonical resources"))
		}
	}
	return findings
}

func safeInputPath(value string) bool {
	if value == "" || strings.ContainsAny(value, `\:*?[]{}`) || strings.ContainsRune(value, 0) || strings.HasPrefix(value, "/") || value == "." || strings.HasPrefix(value, "../") {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func providerName(value string) bool {
	if value == "" || len(value) > 63 || !(value[0] >= 'a' && value[0] <= 'z' || value[0] >= '0' && value[0] <= '9') {
		return false
	}
	if value[len(value)-1] == '-' {
		return false
	}
	for _, char := range value {
		if !(char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '-') {
			return false
		}
	}
	return true
}
