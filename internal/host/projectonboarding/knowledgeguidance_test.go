package projectonboarding

import (
	"strings"
	"testing"
)

func TestKnowledgeGuidanceAppearsInBothNativeProviderSkills(t *testing.T) {
	files, err := renderFiles(Options{Providers: []Provider{Codex, Claude}, DocumentationPath: "docs/markitect/project.md"})
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{".agents/skills", ".claude/skills"} {
		base := root + "/markitect-check/"
		for file, required := range map[string][]string{
			"SKILL.md":                      {"explore the project model", "graph relationships", "read-only", "explicit Manager or whole-project scope"},
			"references/operating-guide.md": {"--knowledge-action graph", "nodes[].id", "--knowledge-action explain", "trace", "history", "--knowledge-scope project", "--briefing-history", "knowledge-mcp", "partial", "human acceptance"},
		} {
			content := fileFor(t, Plan{Files: files}, base+file).Content
			for _, phrase := range required {
				if !strings.Contains(strings.ToLower(content), strings.ToLower(phrase)) {
					t.Errorf("%s%s omitted knowledge guidance %q", base, file, phrase)
				}
			}
		}
	}
}
