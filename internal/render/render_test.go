package render

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func TestGenerateCompanionsKeepTextAndRelativeDependencyLinks(t *testing.T) {
	text := "First paragraph.\n\nSecond paragraph with exact spacing.\n"
	rule := resource("Rule", "docs/team/rules/documentation.yaml", "documentation", "team", core.Spec{Text: "Keep docs clear."})
	skill := resource("Skill", "docs/team/skills/write-docs.yaml", "write-docs", "team", core.Spec{
		Description: "Write docs", Text: text, Rules: []core.Ref{{Name: "documentation"}},
	})
	g := &core.Graph{Resources: map[string]*core.Resource{rule.Key(): rule, skill.Key(): skill}}
	outputs, err := Generate(g)
	if err != nil {
		t.Fatal(err)
	}
	body := string(outputs["docs/team/skills/write-docs.md"])
	if !strings.Contains(body, text) {
		t.Fatalf("companion did not preserve spec.text exactly:\n%s", body)
	}
	if !strings.Contains(body, "[Rule: documentation](../rules/documentation.md)") {
		t.Fatalf("dependency link is absent or not relative: %s", body)
	}
	if !strings.Contains(body, Marker+"; source: docs/team/skills/write-docs.yaml") {
		t.Fatalf("source marker is absent: %s", body)
	}
}

func TestGenerateExplicitTargetsPreservesProviderSettingsAndQuotes(t *testing.T) {
	agent := resource("Agent", "docs/team/agents/review.yaml", "review", "team", core.Spec{
		Description: "Review \"carefully\"", Text: "Review the change.", Providers: core.Providers{
			Codex:  &core.Provider{Model: "gpt-6-sol", Effort: "high", Sandbox: "workspace-write", PermissionMode: "ask", Tools: []string{"read_file", "search"}, DisallowedTools: []string{"shell_exec"}, MaxTurns: 7},
			Claude: &core.Provider{Model: "sonnet", Effort: "high", PermissionMode: "default", Tools: []string{"Read", "Grep"}, DisallowedTools: []string{"Bash"}, MaxTurns: 9},
		},
	})
	skill := resource("Skill", "docs/team/skills/review.yaml", "review", "team", core.Spec{Description: "Use the skill", Text: "Review inputs."})
	project := resource("Project", "markitect.yaml", "sample", "", core.Spec{Targets: []string{"codex", "claude"}})
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
	for _, required := range []string{`description = "Review \"carefully\""`, `model_reasoning_effort = "high"`, `sandbox_mode = "workspace-write"`, "../../docs/team/agents/review.md"} {
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
	for _, required := range []string{"permissionMode: \"default\"", `tools: ["Read", "Grep"]`, `disallowedTools: ["Bash"]`, "maxTurns: 9", "../../docs/team/agents/review.md"} {
		if !strings.Contains(claude, required) {
			t.Errorf("Claude agent missing %q:\n%s", required, claude)
		}
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

func TestGenerateProviderOutputsRequireExplicitTargets(t *testing.T) {
	agent := resource("Agent", "docs/area/agents/review.yaml", "review", "area", core.Spec{Text: "Review."})
	project := resource("Project", "markitect.yaml", "sample", "", core.Spec{})
	g := &core.Graph{Resources: map[string]*core.Resource{agent.Key(): agent}, Project: project}
	outputs, err := Generate(g)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := outputs["docs/area/agents/review.md"]; !ok {
		t.Fatal("missing canonical companion view")
	}
	if len(outputs) != 1 {
		t.Fatalf("provider outputs appeared without explicit targets: %v", outputPaths(outputs))
	}
}

func TestSharedRolePointersRespectSelectedTarget(t *testing.T) {
	project := resource("Project", "markitect.yaml", "sample", "", core.Spec{Targets: []string{"codex"}, ProviderAdapters: &core.ProviderAdapters{RoleRegister: "docs/roles.md"}})
	g := &core.Graph{Resources: map[string]*core.Resource{}, Project: project}
	outputs, err := Generate(g)
	if err != nil {
		t.Fatal(err)
	}
	if outputs[".agents/roles.md"] == nil || outputs[".claude/roles.md"] != nil {
		t.Fatalf("Codex target leaked Claude role pointer: %v", outputPaths(outputs))
	}
	project.Spec.Targets = []string{"claude"}
	outputs, err = Generate(g)
	if err != nil {
		t.Fatal(err)
	}
	if outputs[".claude/roles.md"] == nil || outputs[".agents/roles.md"] != nil {
		t.Fatalf("Claude target leaked Codex role pointer: %v", outputPaths(outputs))
	}
}

func TestGenerateRuleAdaptersOnlyFromExplicitMappings(t *testing.T) {
	first := resource("Rule", "docs/area/rules/first.yaml", "first", "area", core.Spec{Text: "First."})
	second := resource("Rule", "docs/other/rules/second.yaml", "second", "other", core.Spec{Text: "Second."})
	project := resource("Project", "markitect.yaml", "sample", "", core.Spec{
		Targets: []string{"claude"}, RuleAdapters: map[string][]core.Ref{
			"review-context": {{Kind: "Rule", Namespace: "other", Name: "second"}},
		},
	})
	g := &core.Graph{Resources: map[string]*core.Resource{first.Key(): first, second.Key(): second}, Project: project}
	outputs, err := Generate(g)
	if err != nil {
		t.Fatal(err)
	}
	adapter := string(outputs[".claude/rules/review-context.md"])
	if !strings.Contains(adapter, "../../docs/other/rules/second.md") {
		t.Fatalf("explicit adapter does not link to its configured Rule: %s", adapter)
	}
	if outputs[".claude/rules/area-first.md"] != nil || outputs[".claude/rules/other-second.md"] != nil {
		t.Fatalf("adapter paths were inferred from namespaces: %v", outputPaths(outputs))
	}
	project.Spec.RuleAdapters = map[string][]core.Ref{"missing": {{Kind: "Rule", Namespace: "other", Name: "absent"}}}
	if _, err := Generate(g); err == nil || !strings.Contains(err.Error(), "unresolved rule") {
		t.Fatalf("unresolved explicit adapter did not fail: %v", err)
	}
}

func TestGenerateWithOwnersReturnsExplicitOutputOwnership(t *testing.T) {
	rule := resource("Rule", "resources/rules/privacy.yaml", "privacy", "policy", core.Spec{Text: "Keep data private."})
	skill := resource("Skill", "resources/skills/review.yaml", "review", "policy", core.Spec{Text: "Review safely.", Rules: []core.Ref{{Name: "privacy"}}})
	project := resource("Project", "markitect.yaml", "sample", "", core.Spec{
		Targets: []string{"claude"}, RuleAdapters: map[string][]core.Ref{
			"privacy": {{Kind: "Rule", Namespace: "policy", Name: "privacy"}},
		},
	})
	g := &core.Graph{Resources: map[string]*core.Resource{rule.Key(): rule, skill.Key(): skill}, Project: project}
	_, owners, err := GenerateWithOwners(g)
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string][]string{
		"resources/rules/privacy.md":     {"policy/Rule/privacy"},
		"resources/skills/review.md":     {"policy/Skill/review"},
		".claude/skills/review/SKILL.md": {"policy/Skill/review"},
		".claude/rules/privacy.md":       {"policy/Rule/privacy"},
	} {
		got := owners[path]
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("owners[%q] = %v, want %v", path, got, want)
		}
	}
	if _, ok := owners[".claude/rules/privacy.md"]; !ok {
		t.Fatal("explicit adapter output has no ownership metadata")
	}
}

func TestGenerateSkipsImportedResourcesButRendersQualifiedDependencyText(t *testing.T) {
	local := resource("Skill", "docs/area/reviewer.yaml", "reviewer", "area", core.Spec{
		Description: "Review changes.",
		Uses:        []core.Ref{{Package: "policy-set", Namespace: "shared", Kind: "Skill", Name: "policy-review"}},
	})
	imported := resource("Skill", "docs/area/reviewer.yaml", "policy-review", "shared", core.Spec{Text: "Imported source must not be rendered."})
	imported.Package = "policy-set"
	project := resource("Project", "markitect.yaml", "sample", "", core.Spec{
		Targets:  []string{"codex"},
		Packages: []core.PackagePin{{Name: "policy-set", Version: "2.4.1"}},
	})
	g := &core.Graph{Project: project, Resources: map[string]*core.Resource{
		local.GraphKey(): local, imported.GraphKey(): imported,
	}}
	outputs, err := Generate(g)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := outputs[".agents/skills/policy-review/SKILL.md"]; ok {
		t.Fatal("imported Skill received a local provider entrypoint")
	}
	if _, ok := outputs["docs/area/reviewer.md"]; !ok {
		t.Fatal("local companion was lost to an imported source path collision")
	}
	body := string(outputs["docs/area/reviewer.md"])
	for _, want := range []string{"policy-set::shared/Skill/policy-review", "version 2.4.1", "--package policy-set", "--namespace shared", "--kind Skill", "--name policy-review"} {
		if !strings.Contains(body, want) {
			t.Errorf("imported dependency text lacks %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "](docs/") || strings.Contains(body, "policy-review.md") {
		t.Fatalf("imported dependency was rendered as a broken local link:\n%s", body)
	}
}

func TestGenerateRejectsExternalRuleAdapters(t *testing.T) {
	imported := resource("Rule", "rules/privacy.yaml", "privacy", "policy", core.Spec{})
	imported.Package = "policy-set"
	project := resource("Project", "markitect.yaml", "sample", "", core.Spec{
		Targets:      []string{"claude"},
		RuleAdapters: map[string][]core.Ref{"privacy": {{Package: "policy-set", Namespace: "policy", Kind: "Rule", Name: "privacy"}}},
	})
	g := &core.Graph{Project: project, Resources: map[string]*core.Resource{imported.GraphKey(): imported}}
	if _, err := Generate(g); err == nil || !strings.Contains(err.Error(), "external ruleAdapters are not supported") {
		t.Fatalf("external rule adapter error = %v", err)
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
