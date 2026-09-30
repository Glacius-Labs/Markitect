package render

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func TestGenerateCompanionsKeepTextAndRelativeDependencyLinks(t *testing.T) {
	text := "First paragraph.\n\nSecond paragraph with exact spacing.\n"
	rule := resource("Rule", "docs/general/rules/documentation.yaml", "documentation", "cockpit-general", core.Spec{Text: "Keep docs clear."})
	skill := resource("Skill", "docs/general/skills/write-docs.yaml", "write-docs", "cockpit-general", core.Spec{
		Description: "Write docs", Text: text, Rules: []core.Ref{{Name: "documentation"}},
	})
	g := &core.Graph{Resources: map[string]*core.Resource{rule.Key(): rule, skill.Key(): skill}}
	outputs, err := Generate(g)
	if err != nil {
		t.Fatal(err)
	}
	body := string(outputs["docs/general/skills/write-docs.md"])
	if !strings.Contains(body, text) {
		t.Fatalf("companion did not preserve spec.text exactly:\n%s", body)
	}
	if !strings.Contains(body, "[Rule: documentation](../rules/documentation.md)") {
		t.Fatalf("dependency link is absent or not relative: %s", body)
	}
	if !strings.Contains(body, Marker+"; source: docs/general/skills/write-docs.yaml") {
		t.Fatalf("source marker is absent: %s", body)
	}
}

func TestGenerateExplicitTargetsPreservesProviderSettingsAndQuotes(t *testing.T) {
	agent := resource("Agent", "docs/general/agents/review.yaml", "review", "cockpit-general", core.Spec{
		Description: "Review \"carefully\"", Text: "Review the change.", Providers: core.Providers{
			Codex:  &core.Provider{Model: "gpt-6-sol", Effort: "high", Sandbox: "workspace-write", PermissionMode: "ask", Tools: []string{"read_file", "search"}, DisallowedTools: []string{"shell_exec"}, MaxTurns: 7},
			Claude: &core.Provider{Model: "sonnet", Effort: "high", PermissionMode: "default", Tools: []string{"Read", "Grep"}, DisallowedTools: []string{"Bash"}, MaxTurns: 9},
		},
	})
	skill := resource("Skill", "docs/general/skills/review.yaml", "review", "cockpit-general", core.Spec{Description: "Use the skill", Text: "Review inputs."})
	project := resource("Project", "markitect.yaml", "cockpit", "", core.Spec{Profile: "cockpit", Targets: []string{"codex", "claude"}})
	g := &core.Graph{Resources: map[string]*core.Resource{agent.Key(): agent, skill.Key(): skill}, Project: project}
	outputs, err := Generate(g)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{".agents/skills/review/SKILL.md", ".claude/skills/review/SKILL.md", ".codex/agents/review.toml", ".claude/agents/review.md"} {
		if _, ok := outputs[p]; !ok {
			t.Errorf("expected output %s", p)
		}
	}
	codex := string(outputs[".codex/agents/review.toml"])
	for _, required := range []string{`description = "Review \"carefully\""`, `model_reasoning_effort = "high"`, `sandbox_mode = "workspace-write"`, "../../docs/general/agents/review.md"} {
		if !strings.Contains(codex, required) {
			t.Errorf("Codex TOML missing %q:\n%s", required, codex)
		}
	}
	for _, unsupported := range []string{"\neffort =", "permission_mode =", "\ntools =", "disallowed_tools =", "max_turns ="} {
		if strings.Contains(codex, unsupported) {
			t.Errorf("Codex TOML contains unsupported key %q:\n%s", unsupported, codex)
		}
	}
	claude := string(outputs[".claude/agents/review.md"])
	for _, required := range []string{"permissionMode: \"default\"", `tools: ["Read", "Grep"]`, `disallowedTools: ["Bash"]`, "maxTurns: 9", "../../docs/general/agents/review.md"} {
		if !strings.Contains(claude, required) {
			t.Errorf("Claude agent missing %q:\n%s", required, claude)
		}
	}
}

func TestKonfyraCompanionsMatchLegacySkillAndAgentFrontmatter(t *testing.T) {
	text := "Review the source.\n\nPreserve this paragraph."
	skill := resource("Skill", "docs/general/skills/review.yaml", "review", "konfyra-general", core.Spec{Description: "Review carefully", Text: text})
	agent := resource("Agent", "docs/general/agents/reviewer.yaml", "reviewer", "konfyra-general", core.Spec{
		Description: "Reviews source", Text: text, Providers: core.Providers{
			Codex:  &core.Provider{Model: "gpt-6-luna", Effort: "high", Sandbox: "workspace-write"},
			Claude: &core.Provider{Model: "sonnet", Effort: "max", PermissionMode: "acceptEdits", Tools: []string{"Read", "Grep"}, DisallowedTools: []string{"Bash"}, MaxTurns: 4},
		},
	})
	project := resource("Project", "markitect.yaml", "konfyra", "", core.Spec{Profile: "konfyra"})
	g := &core.Graph{Resources: map[string]*core.Resource{skill.Key(): skill, agent.Key(): agent}, Project: project}
	outputs, err := Generate(g)
	if err != nil {
		t.Fatal(err)
	}
	wantSkill := "---\nname: review\ndescription: Review carefully\n---\n<!-- " + Marker + "; source: docs/general/skills/review.yaml -->\n" + text
	if got := string(outputs["docs/general/skills/review.md"]); got != wantSkill {
		t.Fatalf("skill companion differs from legacy format:\n%q\nwant:\n%q", got, wantSkill)
	}
	wantAgent := "---\n{\"claude\":{\"disallowedTools\":[\"Bash\"],\"effort\":\"max\",\"maxTurns\":4,\"model\":\"sonnet\",\"permissionMode\":\"acceptEdits\",\"tools\":[\"Read\",\"Grep\"]},\"codex\":{\"model\":\"gpt-6-luna\",\"model_reasoning_effort\":\"high\",\"sandbox_mode\":\"workspace-write\"},\"description\":\"Reviews source\",\"name\":\"reviewer\"}\n---\n<!-- " + Marker + "; source: docs/general/agents/reviewer.yaml -->\n" + text
	if got := string(outputs["docs/general/agents/reviewer.md"]); got != wantAgent {
		t.Fatalf("agent companion differs from legacy format:\n%q\nwant:\n%q", got, wantAgent)
	}
}

func TestKonfyraAllOtherCompanionsContainOnlyMarkerAndOriginalText(t *testing.T) {
	text := "Keep this body exactly.\n\nAnd this paragraph."
	rule := resource("Rule", "docs/general/rules/policy.yaml", "policy", "konfyra-general", core.Spec{Text: text, Description: "Metadata must not be rendered here", Rules: []core.Ref{{Name: "other"}}})
	project := resource("Project", "markitect.yaml", "konfyra", "", core.Spec{Profile: "konfyra"})
	g := &core.Graph{Resources: map[string]*core.Resource{rule.Key(): rule}, Project: project}
	outputs, err := Generate(g)
	if err != nil {
		t.Fatal(err)
	}
	want := "<!-- " + Marker + "; source: docs/general/rules/policy.yaml -->\n" + text
	if got := string(outputs["docs/general/rules/policy.md"]); got != want {
		t.Fatalf("Konfyra Rule companion changed source body or added metadata:\n%q\nwant:\n%q", got, want)
	}
}

func TestKonfyraSkillRejectsDescriptionsLegacyParserWouldChange(t *testing.T) {
	for _, description := range []string{"", "two\nlines", "phrase: with colon", "\"quoted\"", "leading # comment", "true", "123 numeric"} {
		skill := resource("Skill", "docs/general/skills/entry.yaml", "entry", "konfyra-general", core.Spec{Description: description, Text: "Use me."})
		project := resource("Project", "markitect.yaml", "konfyra", "", core.Spec{Profile: "konfyra"})
		g := &core.Graph{Resources: map[string]*core.Resource{skill.Key(): skill}, Project: project}
		if _, err := Generate(g); err == nil || !strings.Contains(err.Error(), "legacy adapter") {
			t.Errorf("description %q did not produce a compatibility error: %v", description, err)
		}
	}
}

func TestKonfyraDefaultsToCompanionsAndExplicitTargetsOptIn(t *testing.T) {
	agent := resource("Agent", "docs/general/agents/review.yaml", "review", "konfyra-general", core.Spec{Text: "Review.", Providers: core.Providers{
		Codex:  &core.Provider{Model: "model", Effort: "high", Sandbox: "workspace-write"},
		Claude: &core.Provider{Model: "model", Effort: "high", PermissionMode: "default"},
	}})
	project := resource("Project", "markitect.yaml", "konfyra", "", core.Spec{Profile: "konfyra"})
	g := &core.Graph{Resources: map[string]*core.Resource{agent.Key(): agent}, Project: project}
	outputs, err := Generate(g)
	if err != nil {
		t.Fatal(err)
	}
	if len(outputs) != 1 || outputs["docs/general/agents/review.md"] == nil {
		t.Fatalf("expected only companion output, got %#v", outputPaths(outputs))
	}
	project.Spec.Targets = []string{"codex"}
	outputs, err = Generate(g)
	if err != nil {
		t.Fatal(err)
	}
	if outputs[".codex/agents/review.toml"] == nil {
		t.Fatal("explicit target did not opt into provider rendering")
	}
}

func TestGenerateRejectsCaseInsensitiveSourceOutputCollision(t *testing.T) {
	r := resource("Text", "docs/a/note.yaml", "note", "test", core.Spec{Text: "source"})
	collision := resource("Text", "DOCS/A/NOTE.MD", "other", "test", core.Spec{Text: "existing output"})
	g := &core.Graph{Resources: map[string]*core.Resource{r.Key(): r, collision.Key(): collision}}
	if _, err := Generate(g); err == nil || !strings.Contains(err.Error(), "collision") {
		t.Fatalf("expected path collision, got %v", err)
	}
}

func TestCockpitRuleDefaultsExcludeCustomerAreas(t *testing.T) {
	general := resource("Rule", "docs/general/rules/shared.yaml", "shared", "cockpit-general", core.Spec{Text: "general"})
	local := resource("Rule", "docs/consiliari/rules/local.yaml", "local", "cockpit-consiliari", core.Spec{Text: "local"})
	customer := resource("Rule", "docs/customers/acme/rules/private.yaml", "private", "cockpit-acme", core.Spec{Text: "private"})
	project := resource("Project", "markitect.yaml", "cockpit", "", core.Spec{Profile: "cockpit", Targets: []string{"claude"}, Areas: []core.Area{
		{Name: "cockpit-general", Rules: []core.Ref{{Name: "shared"}}},
		{Name: "cockpit-consiliari", Rules: []core.Ref{{Name: "shared", Namespace: "cockpit-general"}, {Name: "local"}}},
		{Name: "cockpit-acme", Rules: []core.Ref{{Name: "private"}}},
	}})
	g := &core.Graph{Resources: map[string]*core.Resource{general.Key(): general, local.Key(): local, customer.Key(): customer}, Project: project}
	outputs, err := Generate(g)
	if err != nil {
		t.Fatal(err)
	}
	if outputs[".claude/rules/general-shared.md"] == nil {
		t.Fatal("missing general rule adapter")
	}
	if outputs[".claude/rules/consiliari-local.md"] == nil {
		t.Fatal("missing Consiliari-owned rule adapter")
	}
	if outputs[".claude/rules/consiliari-shared.md"] != nil {
		t.Fatal("inherited General rule was rendered under a Consiliari global name")
	}
	if outputs[".claude/rules/acme-private.md"] != nil {
		t.Fatal("customer rule was rendered as global adapter")
	}
}

func resource(kind, source, name, namespace string, spec core.Spec) *core.Resource {
	return &core.Resource{Kind: kind, Path: source, Metadata: core.Metadata{Name: name, Namespace: namespace}, Spec: spec}
}

func outputPaths(outputs map[string][]byte) []string {
	paths := make([]string, 0, len(outputs))
	for p := range outputs {
		paths = append(paths, p)
	}
	return paths
}
