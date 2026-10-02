package render

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"go.yaml.in/yaml/v3"
)

const domainMarkdownRoot = "docs/markitect/_domains"

// DomainMarkdownPaths returns deterministic contract views for selected custom
// domains. The built-in AI domain has no project-owned definition source and is
// deliberately omitted.
func DomainMarkdownPaths(g *core.Graph) (map[string]core.DomainDefinition, error) {
	paths := map[string]core.DomainDefinition{}
	if g == nil || g.Project == nil || !hasTarget(g.Project.Spec.Targets, "markdown") || g.Registry == nil {
		return paths, nil
	}
	folded := map[string]string{}
	for _, domain := range g.Registry.Domains() {
		if domain.Name == "" || domain.Path == "" || domain.APIVersion == core.APIVersion {
			continue
		}
		output, err := DomainMarkdownPath(domain)
		if err != nil {
			return nil, err
		}
		if previous, ok := folded[foldPath(output)]; ok {
			return nil, fmt.Errorf("domain contract view path collision: %q and %q", previous, output)
		}
		folded[foldPath(output)] = output
		paths[output] = domain
	}
	return paths, nil
}

// DomainMarkdownPath constructs a safe output path solely from the exact API
// identity. Uppercase and punctuation in a valid version are hex encoded so
// case-insensitive filesystems cannot collapse distinct identities.
func DomainMarkdownPath(domain core.DomainDefinition) (string, error) {
	parts := strings.Split(domain.APIVersion, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || domain.Name == "" || domain.Path == "" {
		return "", fmt.Errorf("domain contract requires name, source path, and group/version apiVersion")
	}
	filename := safeDomainPart(parts[0]) + "." + safeDomainPart(parts[1]) + ".domain.md"
	output := path.Join(domainMarkdownRoot, filename)
	if err := validRepoPath(output); err != nil {
		return "", fmt.Errorf("domain contract output path: %w", err)
	}
	return output, nil
}

func safeDomainPart(value string) string {
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '-' {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "_%02x", c)
		}
	}
	return b.String()
}

func renderDomainContract(domain core.DomainDefinition, target string, g *core.Graph) ([]byte, error) {
	type normalizedDefinition struct {
		APIVersion  string                             `yaml:"apiVersion"`
		Kinds       map[string]core.KindDefinition     `yaml:"kinds"`
		Relations   map[string]core.RelationDefinition `yaml:"relations,omitempty"`
		Constraints []core.ConstraintDefinition        `yaml:"constraints,omitempty"`
	}
	definition := normalizedDefinition{domain.APIVersion, domain.Kinds, domain.Relations, domain.Constraints}
	encoded, err := yaml.Marshal(definition)
	if err != nil {
		return nil, fmt.Errorf("encode normalized domain %s: %w", domain.APIVersion, err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "<!-- %s; source: %s -->\n", Marker, domain.Path)
	fmt.Fprintf(&b, "# Domain contract: %s\n\n", markdownText(domain.Name))
	fmt.Fprintf(&b, "- **API version:** `%s`\n- **Definition source:** `%s`\n\n", markdownCode(domain.APIVersion), markdownCode(domain.Path))
	b.WriteString("## Normalized schema and policy\n\n```yaml\n")
	b.Write(encoded)
	b.WriteString("```\n\n")
	writePolicyOutcomes(&b, domain.APIVersion, "", g, target)
	return []byte(strings.TrimRight(b.String(), "\n") + "\n"), nil
}

func renderDomainIndex(views map[string]core.DomainDefinition, target string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "<!-- %s; source: markitect.yaml -->\n# Domain contracts\n\n", Marker)
	paths := make([]string, 0, len(views))
	for name := range views {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	for _, name := range paths {
		domain := views[name]
		label := fmt.Sprintf("%s (`%s`)", markdownText(domain.Name), markdownCode(domain.APIVersion))
		fmt.Fprintf(&b, "- [%s](%s) — source `%s`\n", label, relative(target, name), markdownCode(domain.Path))
	}
	return []byte(b.String())
}

func markdownCode(value string) string {
	return strings.ReplaceAll(value, "`", "\\`")
}

func markdownText(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	for _, char := range []string{"`", "*", "_", "[", "]", "<", ">", "|"} {
		value = strings.ReplaceAll(value, char, "\\"+char)
	}
	return strings.ReplaceAll(strings.ReplaceAll(value, "\r", " "), "\n", " ")
}
