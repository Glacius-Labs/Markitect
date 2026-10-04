package agentrules

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func TestGenerateProjectsOnlySelectedProvidersAndPreservesCanonicalOwnership(t *testing.T) {
	project := resource("Project", "markitect.yaml", "project", map[string]any{})
	agent := resource("Agent", "docs/agents/reviewer.yaml", "reviewer", map[string]any{
		"description": "Review changes",
		"text":        "Read [the rule](../rules/security.yaml).",
	})
	rule := resource("Rule", "docs/rules/security.yaml", "security", map[string]any{"text": "Protect data."})
	model := core.SemanticModel{Resources: []core.ModelResource{project, agent, rule}, Relationships: []core.ModelRelationship{{From: agent.Identity.Key, To: rule.Identity.Key, Type: "rules"}}}
	config := Config{
		Targets:          []string{"claude"},
		ProjectKey:       project.Identity.Key,
		ProjectPath:      project.Source.Path,
		RuleAdapters:     []RuleAdapter{{Name: "review", RuleKeys: []string{rule.Identity.Key}}},
		AgentSettings:    map[string]AgentSettings{agent.Identity.Key: {Codex: &CodexSettings{Model: "codex-model"}, Claude: &ClaudeSettings{Model: "claude-model", Effort: "high", PermissionMode: "plan", Tools: []string{"Read"}}}},
		ProviderAdapters: ProviderAdapters{InlineAgentText: true},
	}
	outputs, owners, err := Generate(model, config, map[string][]byte{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := outputs[".codex/agents/reviewer.toml"]; ok {
		t.Fatal("Codex output was generated without selecting Codex")
	}
	claudePath := ".claude/agents/reviewer.md"
	if got := string(outputs[claudePath]); !strings.Contains(got, `model: "claude-model"`) || !strings.Contains(got, `permissionMode: "plan"`) || !strings.Contains(got, "[the rule](../../docs/rules/security.yaml)") {
		t.Fatalf("Claude projection did not use selected settings/canonical paths: %s", got)
	}
	if got := string(outputs[".claude/rules/review.md"]); !strings.Contains(got, "[security](../../docs/rules/security.yaml)") {
		t.Fatalf("Rule adapter omitted canonical Rule pointer: %s", got)
	}
	if got, want := owners[claudePath], []string{agent.Identity.Key}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Agent owner = %v, want %v", got, want)
	}
	if got, want := owners[".claude/rules/review.md"], []string{rule.Identity.Key}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Rule adapter owners = %v, want %v", got, want)
	}
	if _, ok := outputs["docs/markitect/agents/reviewer.md"]; ok {
		t.Fatal("Agent Rules module generated a Markdown view")
	}
}

func TestGenerateIsDeterministicAndProviderAdaptersRemainIndependent(t *testing.T) {
	project := resource("Project", "markitect.yaml", "project", map[string]any{})
	agent := resource("Agent", "agents/reviewer.yaml", "reviewer", map[string]any{"description": "Review", "text": "Review."})
	model := core.SemanticModel{Resources: []core.ModelResource{agent, project}}
	config := Config{
		Targets:          []string{"codex", "claude"},
		ProjectKey:       project.Identity.Key,
		ProjectPath:      project.Source.Path,
		AgentSettings:    map[string]AgentSettings{agent.Identity.Key: {Codex: &CodexSettings{Model: "c", Effort: "high", Sandbox: "workspace"}, Claude: &ClaudeSettings{Model: "a", Effort: "medium", PermissionMode: "plan"}}},
		ProviderAdapters: ProviderAdapters{ClaudeRuleSources: map[string][]string{"zeta": {"docs/z.yaml"}, "alpha": {"docs/a.yaml"}}},
	}
	files := map[string][]byte{"docs/z.yaml": []byte("z"), "docs/a.yaml": []byte("a")}
	first, firstOwners, err := Generate(model, config, files)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		next, nextOwners, err := Generate(model, config, files)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(firstOwners, nextOwners) || !equalFiles(first, next) {
			t.Fatal("generation changed across repeated runs")
		}
	}
	if !strings.Contains(string(first[".codex/agents/reviewer.toml"]), `model = "c"`) {
		t.Fatalf("Codex settings were not projected independently: %s", first[".codex/agents/reviewer.toml"])
	}
	if !strings.Contains(string(first[".claude/agents/reviewer.md"]), `model: "a"`) {
		t.Fatalf("Claude settings were not projected independently: %s", first[".claude/agents/reviewer.md"])
	}
	if !bytes.Contains(first[".claude/rules/alpha.md"], []byte("docs/a.yaml")) || !bytes.Contains(first[".claude/rules/zeta.md"], []byte("docs/z.yaml")) {
		t.Fatal("explicit Claude rule sources were not projected")
	}
}

func TestDependenciesComeFromResolvedCoreRelationshipsAndPackagePins(t *testing.T) {
	skill := resource("Skill", "skills/review.yaml", "review", map[string]any{"description": "Review"})
	packageRule := resource("Rule", "packages/guidance/rules/security.yaml", "security", map[string]any{})
	packageRule.Identity.Package = "guidance"
	model := core.SemanticModel{
		Resources: []core.ModelResource{skill, packageRule},
		Relationships: []core.ModelRelationship{{
			From: skill.Identity.Key, To: packageRule.Identity.Key, Type: "rules",
		}},
	}
	config := Config{Targets: []string{"codex"}, PackageVersions: map[string]string{"guidance": "2.4.1"}}
	outputs, _, err := Generate(model, config, nil)
	if err != nil {
		t.Fatal(err)
	}
	body := string(outputs[".agents/skills/review/SKILL.md"])
	for _, value := range []string{"Package dependency: " + packageRule.Identity.Key + " (version 2.4.1)", "--package guidance", "--kind Rule", "--name security"} {
		if !strings.Contains(body, value) {
			t.Errorf("dependency projection lacks %q: %s", value, body)
		}
	}
}

func TestValidateChecksStrictMappingsSettingsAndInventory(t *testing.T) {
	project := resource("Project", "markitect.yaml", "project", map[string]any{})
	agent := resource("Agent", "agents/reviewer.yaml", "reviewer", map[string]any{"description": "Review", "text": "Review."})
	rule := resource("Rule", "rules/security.yaml", "security", map[string]any{})
	model := core.SemanticModel{Resources: []core.ModelResource{project, agent, rule}}
	config := Config{
		Targets:          []string{"codex", "claude"},
		ProjectKey:       project.Identity.Key,
		ProjectPath:      project.Source.Path,
		ProviderAdapters: ProviderAdapters{StrictInventory: true, RetiredSkills: []string{"legacy"}},
	}
	files := map[string][]byte{
		".agents/skills/legacy/SKILL.md": []byte("human-owned legacy"),
		".claude/agents/unregistered.md": []byte("human-owned"),
	}
	findings := Validate(model, config, files)
	codes := map[string]bool{}
	for _, finding := range findings {
		codes[finding.Code] = true
	}
	for _, code := range []string{"provider-adapter.unmapped-rule", "provider-adapter.settings", "retired-output", "unregistered-output"} {
		if !codes[code] {
			t.Errorf("missing %s diagnostic in %+v", code, findings)
		}
	}
}

func TestRoleEntryPointsAreOwnedByProjectAndRequireExplicitInputBytes(t *testing.T) {
	project := resource("Project", "markitect.yaml", "project", map[string]any{})
	model := core.SemanticModel{Resources: []core.ModelResource{project}}
	config := Config{
		Targets:     []string{"codex", "claude"},
		ProjectKey:  project.Identity.Key,
		ProjectPath: project.Source.Path,
		ProviderAdapters: ProviderAdapters{
			RoleRegister: "docs/roles.md",
		},
	}
	outputs, owners, err := Generate(model, config, map[string][]byte{"docs/roles.md": []byte("role register")})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{".agents/roles.md", ".claude/roles.md"} {
		if !strings.Contains(string(outputs[path]), "docs/roles.md") {
			t.Errorf("role pointer %s does not refer to explicit input: %s", path, outputs[path])
		}
		if got, want := owners[path], []string{project.Identity.Key}; !reflect.DeepEqual(got, want) {
			t.Errorf("owner for %s = %v, want %v", path, got, want)
		}
	}
	if findings := Validate(model, config, nil); !hasDiagnostic(findings, "provider-adapter.source") {
		t.Fatalf("missing explicit role register bytes were not rejected: %+v", findings)
	}
}

func TestGenerateRejectsOutputAndSourceCollisions(t *testing.T) {
	project := resource("Project", "markitect.yaml", "project", map[string]any{})
	agent := resource("Agent", ".claude/agents/reviewer.md", "reviewer", map[string]any{"description": "Review"})
	model := core.SemanticModel{Resources: []core.ModelResource{project, agent}}
	config := Config{Targets: []string{"claude"}, ProjectKey: project.Identity.Key, ProjectPath: project.Source.Path}
	if _, _, err := Generate(model, config, nil); err == nil || !strings.Contains(err.Error(), "output path collision") {
		t.Fatalf("expected output/source collision, got %v", err)
	}

	model.Resources[1].Source.Path = "docs/agents/reviewer.yaml"
	duplicate := resource("Agent", "agents/other.yaml", "reviewer", map[string]any{"description": "Duplicate"})
	model.Resources = append(model.Resources, duplicate)
	config.Targets = []string{"claude"}
	if _, _, err := Generate(model, config, nil); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate generated output to fail, got %v", err)
	}
}

func TestIsGeneratedRecognizesProviderMarkerAfterFrontmatter(t *testing.T) {
	for _, data := range [][]byte{
		[]byte("<!-- Generated by Markitect; source: a.yaml -->\n"),
		[]byte("---\nname: x\n---\n\n<!-- Generated by Markitect; source: a.yaml -->\n"),
		[]byte("# Generated by Markitect; source: a.yaml\n"),
	} {
		if !IsGenerated(data) {
			t.Errorf("did not recognize marker in %q", data)
		}
	}
	if IsGenerated([]byte("---\nnot closed\n<!-- Generated by Markitect; -->")) {
		t.Fatal("accepted malformed frontmatter as generated")
	}
}

func resource(kind, source, name string, data map[string]any) core.ModelResource {
	key := kind + ":" + name
	return core.ModelResource{
		Identity: core.ModelIdentity{APIVersion: core.APIVersion, Kind: kind, Name: name, Key: key},
		Data:     data,
		Source:   core.ModelSource{Path: source, Digest: "sha256:test"},
	}
}

func equalFiles(left, right map[string][]byte) bool {
	if len(left) != len(right) {
		return false
	}
	for path, data := range left {
		if !bytes.Equal(data, right[path]) {
			return false
		}
	}
	return true
}

func hasDiagnostic(findings []core.Diagnostic, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}
