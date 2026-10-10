package projectcli

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectonboarding"
)

// TestGeneratedGuidanceNamesOnlyExistingVerbsAndTools fails when the guidance
// Markitect writes into adopting repositories names a CLI verb or MCP tool
// that the verb table does not have.
func TestGeneratedGuidanceNamesOnlyExistingVerbsAndTools(t *testing.T) {
	files, err := projectonboarding.Files(projectonboarding.Options{
		Providers:         []projectonboarding.Provider{projectonboarding.Codex, projectonboarding.Claude},
		DocumentationPath: "docs/markitect/project.md",
	})
	if err != nil {
		t.Fatal(err)
	}
	tools := map[string]bool{}
	for _, v := range verbTable() {
		if v.mcpVerb(false) {
			tools[v.name] = true
		}
	}
	// Inline code that is not a tool: the runtime pins and the generated skills.
	pins := projectonboarding.Pins()
	guidanceWords := map[string]bool{pins.Model: true, pins.Effort: true}
	skill := regexp.MustCompile(`^\.(?:agents|claude)/skills/([a-z0-9-]+)/SKILL\.md$`)
	for _, file := range files {
		if match := skill.FindStringSubmatch(file.Path); match != nil {
			guidanceWords[match[1]] = true
		}
	}
	command := regexp.MustCompile(`(?:\bmarkitect|ABSOLUTE_MARKITECT_EXECUTABLE) ([a-z][a-z-]*)`)
	inline := regexp.MustCompile("`([a-z][a-z0-9-]*)`")
	unknown := map[string][]string{}
	for _, file := range files {
		for _, removed := range []string{"project_", "markitect project", "project mcp"} {
			if strings.Contains(file.Content, removed) {
				t.Errorf("%s names the removed %q", file.Path, removed)
			}
		}
		for _, match := range command.FindAllStringSubmatch(file.Content, -1) {
			if _, ok := lookupVerb(match[1]); !ok {
				unknown[match[1]] = append(unknown[match[1]], file.Path)
			}
		}
		for _, match := range inline.FindAllStringSubmatch(file.Content, -1) {
			if !tools[match[1]] && !guidanceWords[match[1]] {
				unknown[match[1]] = append(unknown[match[1]], file.Path)
			}
		}
	}
	names := make([]string, 0, len(unknown))
	for name := range unknown {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		t.Errorf("generated guidance names %q, which is neither a verb nor an MCP tool (in %s)", name, strings.Join(unknown[name], ", "))
	}
}
