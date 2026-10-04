package agentrules

import (
	"fmt"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func renderSkill(resource core.ModelResource, target string, model core.SemanticModel, config Config, resources map[string]core.ModelResource) []byte {
	var b strings.Builder
	name, description := resource.Identity.Name, dataString(resource.Data, "description")
	b.WriteString("---\nname: " + yamlQuote(name) + "\ndescription: " + yamlQuote(description) + "\n---\n\n")
	b.WriteString("<!-- " + Marker + "; source: " + relative(target, resource.Source.Path) + " -->\n\n")
	b.WriteString("Read the [canonical Skill YAML](" + relative(target, resource.Source.Path) + ") before acting.\n")
	links := dependencyLinks(resource, target, model, config, resources)
	if len(links) > 0 {
		b.WriteString("\n## Dependencies\n\n")
		for _, link := range links {
			writeDependency(&b, link)
		}
	}
	return []byte(b.String())
}

func renderCodexAgent(resource core.ModelResource, target string, config Config, settings *CodexSettings, resources map[string]core.ModelResource, files map[string][]byte) ([]byte, error) {
	name, description := resource.Identity.Name, dataString(resource.Data, "description")
	var b strings.Builder
	b.WriteString("# " + Marker + "; source: " + relative(target, resource.Source.Path) + "\n")
	if settings != nil {
		writeTOMLString(&b, "name", name)
		writeTOMLString(&b, "description", description)
		if settings.Model != "" {
			writeTOMLString(&b, "model", settings.Model)
		}
		if settings.Effort != "" {
			writeTOMLString(&b, "model_reasoning_effort", settings.Effort)
		}
		if settings.Sandbox != "" {
			writeTOMLString(&b, "sandbox_mode", settings.Sandbox)
		}
	}
	instructions := "Read the canonical [Agent YAML](" + relative(target, resource.Source.Path) + ") and follow its stated scope."
	if config.ProviderAdapters.InlineAgentText {
		text, err := rewriteAgentText(dataString(resource.Data, "text"), resource, target, resources, files)
		if err != nil {
			return nil, err
		}
		instructions = strings.TrimSpace(text)
	}
	if config.ProviderAdapters.AgentContract != "" {
		instructions += "\n\nShared role-routing and delegation rules: [project agent contract](" + relative(target, config.ProviderAdapters.AgentContract) + ")."
	}
	if config.ProviderAdapters.InlineAgentText {
		instructions += "\n"
	}
	writeTOMLString(&b, "developer_instructions", instructions)
	return []byte(b.String()), nil
}

func renderClaudeAgent(resource core.ModelResource, target string, config Config, settings *ClaudeSettings, model core.SemanticModel, resources map[string]core.ModelResource, files map[string][]byte) ([]byte, error) {
	name, description := resource.Identity.Name, dataString(resource.Data, "description")
	var b strings.Builder
	b.WriteString("---\nname: " + yamlQuote(name) + "\ndescription: " + yamlQuote(description) + "\n")
	if settings != nil {
		if settings.Model != "" {
			b.WriteString("model: " + yamlQuote(settings.Model) + "\n")
		}
		if settings.Effort != "" {
			b.WriteString("effort: " + yamlQuote(settings.Effort) + "\n")
		}
		if settings.PermissionMode != "" {
			b.WriteString("permissionMode: " + yamlQuote(settings.PermissionMode) + "\n")
		}
		if len(settings.Tools) > 0 {
			b.WriteString("tools: " + yamlArray(settings.Tools) + "\n")
		}
		if len(settings.DisallowedTools) > 0 {
			b.WriteString("disallowedTools: " + yamlArray(settings.DisallowedTools) + "\n")
		}
		if settings.MaxTurns > 0 {
			fmt.Fprintf(&b, "maxTurns: %d\n", settings.MaxTurns)
		}
	}
	b.WriteString("---\n\n<!-- " + Marker + "; source: " + relative(target, resource.Source.Path) + " -->\n\n")
	if config.ProviderAdapters.InlineAgentText {
		text, err := rewriteAgentText(dataString(resource.Data, "text"), resource, target, resources, files)
		if err != nil {
			return nil, err
		}
		b.WriteString(strings.TrimSpace(text) + "\n")
	} else {
		b.WriteString("Read the canonical [Agent YAML](" + relative(target, resource.Source.Path) + ") before acting.\n")
	}
	if config.ProviderAdapters.AgentContract != "" {
		b.WriteString("\nShared role-routing and delegation rules: [project agent contract](" + relative(target, config.ProviderAdapters.AgentContract) + ").\n")
	}
	links := dependencyLinks(resource, target, model, config, resources)
	if len(links) > 0 {
		b.WriteString("\n## Dependencies\n\n")
		for _, link := range links {
			writeDependency(&b, link)
		}
	}
	return []byte(b.String()), nil
}

type dependencyLink struct {
	label       string
	path        string
	instruction string
}

func dependencyLinks(resource core.ModelResource, target string, model core.SemanticModel, config Config, resources map[string]core.ModelResource) []dependencyLink {
	seen := map[string]bool{}
	var links []dependencyLink
	for _, relationship := range model.Relationships {
		if relationship.From != resource.Identity.Key || !dependencyRelation(relationship.Type) || seen[relationship.To] {
			continue
		}
		seen[relationship.To] = true
		dependency, ok := resources[relationship.To]
		if !ok || dependency.Identity.Kind == "Project" || dependency.Source.Path == "" {
			continue
		}
		if dependency.Identity.Package != "" {
			version := config.PackageVersions[dependency.Identity.Package]
			label := "Package dependency: " + dependency.Identity.Key
			if version != "" {
				label += " (version " + version + ")"
			}
			instruction := "Select this exported dependency with `markitect context --repo . --package " + dependency.Identity.Package + " --namespace " + dependency.Identity.Namespace + " --kind " + dependency.Identity.Kind + " --name " + dependency.Identity.Name + "`."
			links = append(links, dependencyLink{label: label, instruction: instruction})
			continue
		}
		links = append(links, dependencyLink{label: dependency.Identity.Kind + ": " + dependency.Identity.Name, path: relative(target, dependency.Source.Path)})
	}
	sort.Slice(links, func(i, j int) bool {
		if links[i].label != links[j].label {
			return links[i].label < links[j].label
		}
		if links[i].path != links[j].path {
			return links[i].path < links[j].path
		}
		return links[i].instruction < links[j].instruction
	})
	return links
}

func dependencyRelation(name string) bool {
	switch name {
	case "rules", "uses", "needs", "implements":
		return true
	default:
		return false
	}
}

func writeDependency(b *strings.Builder, link dependencyLink) {
	if link.instruction != "" {
		b.WriteString("- " + link.label + ". " + link.instruction + "\n")
		return
	}
	b.WriteString("- [" + link.label + "](" + link.path + ")\n")
}

func dataString(data map[string]any, key string) string {
	value, _ := data[key].(string)
	return value
}

func yamlQuote(value string) string { return strconv.Quote(value) }

func yamlArray(values []string) string {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = yamlQuote(value)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func writeTOMLString(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "%s = %s\n", key, strconv.Quote(value))
}

func relative(fromFile, toFile string) string {
	return relativePath(path.Dir(fromFile), toFile)
}

func relativePath(fromDir, to string) string {
	from := strings.Split(strings.Trim(fromDir, "/"), "/")
	toParts := strings.Split(strings.Trim(to, "/"), "/")
	if len(from) == 1 && from[0] == "" {
		from = nil
	}
	i := 0
	for i < len(from) && i < len(toParts) && from[i] == toParts[i] {
		i++
	}
	parts := make([]string, 0, len(from)-i+len(toParts)-i)
	for range from[i:] {
		parts = append(parts, "..")
	}
	parts = append(parts, toParts[i:]...)
	if len(parts) == 0 {
		return "."
	}
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}
