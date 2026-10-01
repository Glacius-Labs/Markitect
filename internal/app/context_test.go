package app

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestCompileContextIncludesOnlySelectedImplementationAndItsExplicitFiles(t *testing.T) {
	project := core.Resource{
		APIVersion: core.APIVersion,
		Kind:       "Project",
		Metadata:   core.Metadata{Name: "sample"},
		Path:       projectPath,
		Spec: core.Spec{
			Areas: []core.Area{{Name: projectNS, Path: "docs/area"}},
			Bindings: []core.Binding{{
				Contract:       core.Ref{Kind: "Contract", Name: "review", Namespace: projectNS},
				Implementation: core.Ref{Kind: "Agent", Name: "reviewer-a", Namespace: projectNS},
			}},
		},
	}
	contract := core.Resource{
		APIVersion: core.APIVersion, Kind: "Contract", Metadata: core.Metadata{Name: "review", Namespace: projectNS},
		Path: "docs/area/contracts/review.yaml",
		Spec: core.Spec{Text: "Review contract.", Kind: "Agent", Input: []string{"change"}, Output: []string{"findings"}, Files: []string{"docs/area/contracts/review.go"}},
	}
	selected := core.Resource{
		APIVersion: core.APIVersion, Kind: "Agent", Metadata: core.Metadata{Name: "reviewer-a", Namespace: projectNS},
		Path: "docs/area/agents/reviewer-a.yaml",
		Spec: core.Spec{Text: "Selected reviewer.", Input: []string{"change"}, Output: []string{"findings"}, Implements: []core.Ref{{Name: "review"}}, Files: []string{"docs/area/agents/selected-notes.md"}},
	}
	alternative := core.Resource{
		APIVersion: core.APIVersion, Kind: "Agent", Metadata: core.Metadata{Name: "reviewer-b", Namespace: projectNS},
		Path: "docs/area/agents/reviewer-b.yaml",
		Spec: core.Spec{Text: "Alternative reviewer.", Input: []string{"change"}, Output: []string{"findings"}, Implements: []core.Ref{{Name: "review"}}, Files: []string{"docs/area/agents/alternative-notes.md"}},
	}
	consumer := core.Resource{
		APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "review-change", Namespace: projectNS},
		Path: "docs/area/workflows/review-change.yaml",
		Spec: core.Spec{Text: "Review the change.", Needs: []core.Ref{{Name: "review"}}},
	}

	resources := []*core.Resource{&project, &contract, &selected, &alternative, &consumer}
	snapshot := &snapshot.Snapshot{ID: "fixed-review-snapshot", Files: map[string][]byte{}, Modes: map[string]string{}}
	for _, resource := range resources {
		data := encodeResource(t, *resource)
		snapshot.Files[resource.Path] = data
		snapshot.Modes[resource.Path] = "100644"
	}
	texts := map[string]string{
		"docs/area/contracts/review.go":         "type ReviewInput struct{}\n",
		"docs/area/agents/selected-notes.md":    "Selected reference text.\n",
		"docs/area/agents/alternative-notes.md": "Unselected reference text.\n",
	}
	for name, text := range texts {
		snapshot.Files[name] = []byte(text)
		snapshot.Modes[name] = "100644"
	}

	parsed, err := Parse(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Diagnostics) != 0 {
		t.Fatalf("unexpected graph/input diagnostics: %#v", parsed.Diagnostics)
	}
	ctx, err := CompileContext(parsed, consumer.Metadata.Namespace+"/Workflow/"+consumer.Metadata.Name, "test", "tool-digest")
	if err != nil {
		t.Fatal(err)
	}
	if ctx.SnapshotDigest != snapshot.Digest() || ctx.ToolDigest != "tool-digest" || ctx.Provisional {
		t.Fatalf("context is not bound to the fixed snapshot/tool: %#v", ctx)
	}
	entries := make(map[string]ContextInput, len(ctx.Inputs))
	for _, input := range ctx.Inputs {
		entries[input.Key] = input
	}
	for _, key := range []string{contract.Key(), selected.Key(), consumer.Key(), project.Key()} {
		if _, ok := entries[key]; !ok {
			t.Errorf("context omitted required resource %s", key)
		}
	}
	if _, ok := entries[alternative.Key()]; ok {
		t.Errorf("context included unselected alternative %s", alternative.Key())
	}
	for name, want := range texts {
		key := "file:" + name
		entry, ok := entries[key]
		if name == "docs/area/agents/alternative-notes.md" {
			if ok {
				t.Errorf("context included unselected input file %s", name)
			}
			continue
		}
		if !ok {
			t.Errorf("context omitted declared input file %s", name)
			continue
		}
		if entry.Text != want || entry.Hash != Hash([]byte(want)) {
			t.Errorf("file context content/hash mismatch for %s: %#v", name, entry)
		}
	}
}

func TestChangesMapsSkillSourceAndCompanionToLocalDependents(t *testing.T) {
	makeProject := func(body, companion string) *Project {
		policy := impactProjectResource([]core.Area{{Name: "area", Path: "docs/area"}})
		skill := core.Resource{APIVersion: core.APIVersion, Kind: "Skill", Metadata: core.Metadata{Name: "local", Namespace: "area"}, Path: "docs/area/skills/local.yaml", Spec: core.Spec{Text: body}}
		consumer := core.Resource{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "uses-local", Namespace: "area"}, Path: "docs/area/workflows/uses-local.yaml", Spec: core.Spec{Text: "Use local skill.", Uses: []core.Ref{{Kind: "Skill", Name: "local"}}}}
		unrelated := core.Resource{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "unrelated", Namespace: "area"}, Path: "docs/area/workflows/unrelated.yaml", Spec: core.Spec{Text: "Independent."}}
		return parseImpactProject(t, []*core.Resource{policy, &skill, &consumer, &unrelated}, map[string]string{"docs/area/skills/local.md": companion})
	}
	impact := Changes(makeProject("Old definition.", "Old generated view."), makeProject("New definition.", "New generated view."))
	assertAffected(t, impact, "area/Skill/local", "area/Workflow/uses-local")
}

func TestChangesMapsAgentMetadataAndProviderOutputsToDependents(t *testing.T) {
	makeProject := func(model, codexOutput, claudeOutput string) *Project {
		policy := impactProjectResource([]core.Area{{Name: "area", Path: "docs/area"}})
		policy.Spec.Targets = []string{"codex", "claude"}
		agent := core.Resource{APIVersion: core.APIVersion, Kind: "Agent", Metadata: core.Metadata{Name: "reviewer", Namespace: "area"}, Path: "docs/area/agents/reviewer.yaml", Spec: core.Spec{Text: "Review changes.", Providers: core.Providers{Codex: &core.Provider{Model: model}}}}
		consumer := core.Resource{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "review", Namespace: "area"}, Path: "docs/area/workflows/review.yaml", Spec: core.Spec{Text: "Review.", Uses: []core.Ref{{Kind: "Agent", Name: "reviewer"}}}}
		unrelated := core.Resource{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "other", Namespace: "area"}, Path: "docs/area/workflows/other.yaml", Spec: core.Spec{Text: "Independent."}}
		outputs := map[string]string{
			".codex/agents/reviewer.toml": codexOutput,
			".claude/agents/reviewer.md":  claudeOutput,
		}
		return parseImpactProject(t, []*core.Resource{policy, &agent, &consumer, &unrelated}, outputs)
	}
	impact := Changes(makeProject("old-model", "old codex output", "old claude output"), makeProject("new-model", "new codex output", "new claude output"))
	assertAffected(t, impact, "area/Agent/reviewer", "area/Workflow/review")
}

func TestChangesMapsSharedRuleSourceViewsAndProviderViewToAllApplicableResources(t *testing.T) {
	makeProject := func(ruleText, companion, providerView string) *Project {
		policy := impactProjectResource([]core.Area{{Name: "team", Path: "docs/team"}})
		policy.Spec.Targets = []string{"claude"}
		policy.Spec.RuleAdapters = map[string][]core.Ref{"review-context": {{Kind: "Rule", Namespace: "team", Name: "shared"}}}
		rule := core.Resource{APIVersion: core.APIVersion, Kind: "Rule", Metadata: core.Metadata{Name: "shared", Namespace: "team"}, Path: "docs/team/rules/shared.yaml", Spec: core.Spec{Text: ruleText}}
		area := core.Resource{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "area-flow", Namespace: "team"}, Path: "docs/team/workflows/area-flow.yaml", Spec: core.Spec{Text: "General.", Rules: []core.Ref{{Name: "shared"}}}}
		second := core.Resource{APIVersion: core.APIVersion, Kind: "Skill", Metadata: core.Metadata{Name: "area-skill", Namespace: "team"}, Path: "docs/team/skills/area-skill.yaml", Spec: core.Spec{Text: "General skill.", Rules: []core.Ref{{Name: "shared"}}}}
		outputs := map[string]string{
			"docs/team/rules/shared.md":       companion,
			".claude/rules/review-context.md": providerView,
		}
		return parseImpactProject(t, []*core.Resource{policy, &rule, &area, &second}, outputs)
	}
	impact := Changes(makeProject("Old policy.", "Old rule view.", "Old provider rule."), makeProject("New policy.", "New rule view.", "New provider rule."))
	assertAffected(t, impact, "/Project/impact-test", "team/Rule/shared", "team/Workflow/area-flow", "team/Skill/area-skill")
}

func TestChangesIncludesRemovedResourceFromOldGraph(t *testing.T) {
	policy := impactProjectResource([]core.Area{{Name: "area", Path: "docs/area"}})
	skill := core.Resource{APIVersion: core.APIVersion, Kind: "Skill", Metadata: core.Metadata{Name: "retired", Namespace: "area"}, Path: "docs/area/skills/retired.yaml", Spec: core.Spec{Text: "Retired."}}
	consumer := core.Resource{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "consumer", Namespace: "area"}, Path: "docs/area/workflows/consumer.yaml", Spec: core.Spec{Text: "Consumer."}}
	before := parseImpactProject(t, []*core.Resource{policy, &skill, &consumer}, nil)
	afterPolicy := impactProjectResource([]core.Area{{Name: "area", Path: "docs/area"}})
	afterConsumer := consumer
	after := parseImpactProject(t, []*core.Resource{afterPolicy, &afterConsumer}, nil)
	impact := Changes(before, after)
	if !contains(impact.Affected, skill.Key()) {
		t.Fatalf("impact omitted resource removed from the old graph %s: %v", skill.Key(), impact.Affected)
	}
}

func impactProjectResource(areas []core.Area) *core.Resource {
	return &core.Resource{APIVersion: core.APIVersion, Kind: "Project", Metadata: core.Metadata{Name: "impact-test"}, Path: projectPath, Spec: core.Spec{Areas: areas}}
}

func parseImpactProject(t *testing.T, resources []*core.Resource, extraFiles map[string]string) *Project {
	t.Helper()
	snapshot := &snapshot.Snapshot{ID: "impact-test", Files: map[string][]byte{}, Modes: map[string]string{}}
	for _, resource := range resources {
		data := encodeResource(t, *resource)
		snapshot.Files[resource.Path] = data
		snapshot.Modes[resource.Path] = "100644"
	}
	for name, contents := range extraFiles {
		snapshot.Files[name] = []byte(contents)
		snapshot.Modes[name] = "100644"
	}
	parsed, err := Parse(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Diagnostics) > 0 {
		t.Fatalf("invalid impact fixture: %#v", parsed.Diagnostics)
	}
	return parsed
}

func assertAffected(t *testing.T, impact *Impact, want ...string) {
	t.Helper()
	got := map[string]bool{}
	for _, key := range impact.Affected {
		got[key] = true
	}
	for _, key := range want {
		if !got[key] {
			t.Errorf("impact omitted %s; affected=%v", key, impact.Affected)
		}
		delete(got, key)
	}
	if len(got) > 0 {
		t.Errorf("impact included unrelated resources %v; full affected=%v", mapKeys(got), impact.Affected)
	}
}

func mapKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
