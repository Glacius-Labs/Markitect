package migrate

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/source"
)

func TestKonfyraPlansTypedResourcesAndPreservesMetadataAndBodies(t *testing.T) {
	if !quotedWorkflowPaths.MatchString("Follow `docs/general/workflows/respond.md`.") {
		t.Fatalf("quoted workflow matcher does not match the canonical path: %s", quotedWorkflowPaths.String())
	}
	snapshot := &source.Snapshot{Files: map[string][]byte{
		"docs/general/rules/writing.md":       []byte("# Writing\r\nKeep it clear.\r\n"),
		"docs/general/workflows/respond.md":   []byte("# Respond\n1. Read.\n"),
		"docs/general/skills/respond.md":      []byte("---\r\nname: respond\r\ndescription: \"Route a response.\"\r\n---\r\n\r\n# Skill body\r\n\r\nUse [the response workflow](../workflows/respond.md#steps).\r\n"),
		"docs/general/agents/helper.md":       []byte("---\n{\"name\":\"helper\",\"description\":\"Review a response.\",\"codex\":{\"model\":\"gpt-6-sol\",\"model_reasoning_effort\":\"high\",\"sandbox_mode\":\"read-only\"},\"claude\":{\"model\":\"sonnet\",\"effort\":\"high\",\"permissionMode\":\"plan\",\"tools\":[\"Read\"],\"disallowedTools\":[\"Bash\"],\"maxTurns\":4}}\n---\n\n# Agent body\nFollow `docs/general/workflows/respond.md`.\n"),
		"docs/products/Acme/README.md":        []byte("# Acme\n"),
		"docs/products/Acme/agents/worker.md": []byte("---\n{\"name\":\"worker\",\"description\":\"Work for Acme.\",\"codex\":{\"model\":\"gpt-6-sol\",\"model_reasoning_effort\":\"medium\",\"sandbox_mode\":\"workspace-write\"},\"claude\":{\"model\":\"sonnet\",\"effort\":\"medium\",\"permissionMode\":\"acceptEdits\"}}\n---\nDo the work.\n"),
		"docs/general/skills/README.md":       []byte("not a mechanism"),
	}}

	outputs, err := Konfyra(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(outputs) != 6 {
		t.Fatalf("got %d outputs, want 5 resources plus project: %v", len(outputs), outputNames(outputs))
	}
	if err := ValidateKonfyraParity(snapshot, outputs); err != nil {
		t.Fatal(err)
	}

	resources := make([]*core.Resource, 0, len(outputs))
	for file, data := range outputs {
		resource, err := format.Parse(file, data)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		resources = append(resources, resource)
	}
	graph := core.Build(resources)
	if len(graph.Diagnostics) != 0 {
		t.Fatalf("migrated project graph has diagnostics: %#v", graph.Diagnostics)
	}

	skill, err := format.Parse("docs/general/skills/respond.yaml", outputs["docs/general/skills/respond.yaml"])
	if err != nil {
		t.Fatal(err)
	}
	wantBody := "\n# Skill body\n\nUse [the response workflow](../workflows/respond.md#steps).\n"
	if skill.Spec.Text != wantBody {
		t.Fatalf("skill body changed\n got: %q\nwant: %q", skill.Spec.Text, wantBody)
	}
	if len(skill.Spec.Uses) != 0 {
		t.Fatalf("navigation candidate became an unreviewed dependency: %#v", skill.Spec.Uses)
	}

	agent, err := format.Parse("docs/general/agents/helper.yaml", outputs["docs/general/agents/helper.yaml"])
	if err != nil {
		t.Fatal(err)
	}
	claude := agent.Spec.Providers.Claude
	if claude == nil || claude.Model != "sonnet" || claude.PermissionMode != "plan" || claude.MaxTurns != 4 || strings.Join(claude.Tools, ",") != "Read" || strings.Join(claude.DisallowedTools, ",") != "Bash" {
		t.Fatalf("Claude settings were not preserved: %#v", claude)
	}
	if agent.Metadata.Namespace != "general" || agent.Spec.Providers.Codex == nil || agent.Spec.Providers.Codex.Effort != "high" {
		t.Fatalf("agent ownership or Codex settings were not preserved: %#v", agent)
	}
	if len(agent.Spec.Uses) != 0 {
		t.Fatalf("quoted navigation candidate became an unreviewed dependency: %#v", agent.Spec.Uses)
	}
	candidates, err := CandidateDependencies(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 || candidates[0].Target != "docs/general/workflows/respond.md" || candidates[1].Target != "docs/general/workflows/respond.md" {
		t.Fatalf("workflow navigation candidates are incomplete: %#v", candidates)
	}
	if candidates[0].Context == "" || candidates[1].Context == "" {
		t.Fatalf("workflow candidates lack review context: %#v", candidates)
	}
	rule, err := format.Parse("docs/general/rules/writing.yaml", outputs["docs/general/rules/writing.yaml"])
	if err != nil {
		t.Fatal(err)
	}
	if rule.Spec.Text != "# Writing\nKeep it clear.\n" {
		t.Fatalf("rule body line endings were not normalized: %q", rule.Spec.Text)
	}
	worker, err := format.Parse("docs/products/Acme/agents/worker.yaml", outputs["docs/products/Acme/agents/worker.yaml"])
	if err != nil {
		t.Fatal(err)
	}
	if worker.Metadata.Namespace != "acme" {
		t.Fatalf("product area was not assigned by its longer path: %q", worker.Metadata.Namespace)
	}
}

func TestKonfyraRejectsUnknownFrontmatterFields(t *testing.T) {
	tests := []struct {
		name string
		path string
		data string
	}{
		{
			name: "skill field",
			path: "docs/general/skills/respond.md",
			data: "---\nname: respond\ndescription: route\nallowed-tools: [Bash]\n---\nBody.\n",
		},
		{
			name: "agent provider field",
			path: "docs/general/agents/helper.md",
			data: "---\n{\"name\":\"helper\",\"description\":\"Review.\",\"codex\":{\"model\":\"m\",\"model_reasoning_effort\":\"high\",\"sandbox_mode\":\"read-only\",\"extra\":true},\"claude\":{\"model\":\"m\",\"effort\":\"high\",\"permissionMode\":\"plan\"}}\n---\nBody.\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := &source.Snapshot{Files: map[string][]byte{test.path: []byte(test.data)}}
			if _, err := Konfyra(snapshot); err == nil {
				t.Fatal("expected unsupported metadata to fail")
			}
		})
	}
}

func outputNames(outputs map[string][]byte) []string {
	names := make([]string, 0, len(outputs))
	for name := range outputs {
		names = append(names, name)
	}
	return names
}
