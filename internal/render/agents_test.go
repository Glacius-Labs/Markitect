package render

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func TestAgentContractIsLinkedWithAndWithoutInlineText(t *testing.T) {
	agent := resource("Agent", "docs/agents/reviewer.yaml", "reviewer", "docs", core.Spec{
		Description: "Review changes", Text: "Review the change.",
	})
	project := resource("Project", "markitect.yaml", "sample", "", core.Spec{Targets: []string{"codex", "claude"}})
	g := &core.Graph{Resources: map[string]*core.Resource{agent.Key(): agent}, Project: project}
	for _, inline := range []bool{false, true} {
		project.Spec.ProviderAdapters = &core.ProviderAdapters{AgentContract: "docs/agents/contract.md", InlineAgentText: inline}
		outputs, err := Generate(g)
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{".codex/agents/reviewer.toml", ".claude/agents/reviewer.md"} {
			body := string(outputs[path])
			if !strings.Contains(body, "[project agent contract](../../docs/agents/contract.md)") {
				t.Errorf("inline=%v: %s has no shared contract pointer:\n%s", inline, path, body)
			}
			if inline && !strings.Contains(body, "Review the change.") {
				t.Errorf("inline=%v: %s omitted canonical text:\n%s", inline, path, body)
			}
			if !inline && !strings.Contains(body, "../../docs/agents/reviewer.yaml") {
				t.Errorf("inline=%v: %s omitted canonical companion link:\n%s", inline, path, body)
			}
		}
	}
}
