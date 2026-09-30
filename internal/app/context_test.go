package app

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/source"
)

func TestCompileContextIncludesOnlySelectedImplementationAndItsExplicitFiles(t *testing.T) {
	project := core.Resource{
		APIVersion: core.APIVersion,
		Kind:       "Project",
		Metadata:   core.Metadata{Name: "sample"},
		Path:       projectPath,
		Spec: core.Spec{
			Profile: "generic",
			Areas:   []core.Area{{Name: projectNS, Path: "docs/general"}},
			Bindings: []core.Binding{{
				Contract:       core.Ref{Kind: "Contract", Name: "review", Namespace: projectNS},
				Implementation: core.Ref{Kind: "Agent", Name: "reviewer-a", Namespace: projectNS},
			}},
		},
	}
	contract := core.Resource{
		APIVersion: core.APIVersion, Kind: "Contract", Metadata: core.Metadata{Name: "review", Namespace: projectNS},
		Path: "docs/general/contracts/review.yaml",
		Spec: core.Spec{Text: "Review contract.", Kind: "Agent", Input: []string{"change"}, Output: []string{"findings"}, Files: []string{"docs/general/contracts/review.go"}},
	}
	selected := core.Resource{
		APIVersion: core.APIVersion, Kind: "Agent", Metadata: core.Metadata{Name: "reviewer-a", Namespace: projectNS},
		Path: "docs/general/agents/reviewer-a.yaml",
		Spec: core.Spec{Text: "Selected reviewer.", Input: []string{"change"}, Output: []string{"findings"}, Implements: []core.Ref{{Name: "review"}}, Files: []string{"docs/general/agents/selected-notes.md"}},
	}
	alternative := core.Resource{
		APIVersion: core.APIVersion, Kind: "Agent", Metadata: core.Metadata{Name: "reviewer-b", Namespace: projectNS},
		Path: "docs/general/agents/reviewer-b.yaml",
		Spec: core.Spec{Text: "Alternative reviewer.", Input: []string{"change"}, Output: []string{"findings"}, Implements: []core.Ref{{Name: "review"}}, Files: []string{"docs/general/agents/alternative-notes.md"}},
	}
	consumer := core.Resource{
		APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "review-change", Namespace: projectNS},
		Path: "docs/general/workflows/review-change.yaml",
		Spec: core.Spec{Text: "Review the change.", Needs: []core.Ref{{Name: "review"}}},
	}

	resources := []*core.Resource{&project, &contract, &selected, &alternative, &consumer}
	snapshot := &source.Snapshot{Revision: "fixed-review-snapshot", Files: map[string][]byte{}, Modes: map[string]string{}}
	for _, resource := range resources {
		data := encodeResource(t, *resource)
		snapshot.Files[resource.Path] = data
		snapshot.Modes[resource.Path] = "100644"
	}
	texts := map[string]string{
		"docs/general/contracts/review.go":         "type ReviewInput struct{}\n",
		"docs/general/agents/selected-notes.md":    "Selected reference text.\n",
		"docs/general/agents/alternative-notes.md": "Unselected reference text.\n",
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
		if name == "docs/general/agents/alternative-notes.md" {
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
		policy := impactProjectResource("generic", []core.Area{{Name: "general", Path: "docs/general"}})
		skill := core.Resource{APIVersion: core.APIVersion, Kind: "Skill", Metadata: core.Metadata{Name: "local", Namespace: "general"}, Path: "docs/general/skills/local.yaml", Spec: core.Spec{Text: body}}
		consumer := core.Resource{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "uses-local", Namespace: "general"}, Path: "docs/general/workflows/uses-local.yaml", Spec: core.Spec{Text: "Use local skill.", Uses: []core.Ref{{Kind: "Skill", Name: "local"}}}}
		unrelated := core.Resource{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "unrelated", Namespace: "general"}, Path: "docs/general/workflows/unrelated.yaml", Spec: core.Spec{Text: "Independent."}}
		return parseImpactProject(t, []*core.Resource{policy, &skill, &consumer, &unrelated}, map[string]string{"docs/general/skills/local.md": companion})
	}
	impact := Changes(makeProject("Old definition.", "Old generated view."), makeProject("New definition.", "New generated view."))
	assertAffected(t, impact, "general/Skill/local", "general/Workflow/uses-local")
}

func TestChangesMapsAgentMetadataAndProviderOutputsToDependents(t *testing.T) {
	makeProject := func(model, codexOutput, claudeOutput string) *Project {
		policy := impactProjectResource("generic", []core.Area{{Name: "general", Path: "docs/general"}})
		policy.Spec.Targets = []string{"codex", "claude"}
		agent := core.Resource{APIVersion: core.APIVersion, Kind: "Agent", Metadata: core.Metadata{Name: "reviewer", Namespace: "general"}, Path: "docs/general/agents/reviewer.yaml", Spec: core.Spec{Text: "Review changes.", Providers: core.Providers{Codex: &core.Provider{Model: model}}}}
		consumer := core.Resource{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "review", Namespace: "general"}, Path: "docs/general/workflows/review.yaml", Spec: core.Spec{Text: "Review.", Uses: []core.Ref{{Kind: "Agent", Name: "reviewer"}}}}
		unrelated := core.Resource{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "other", Namespace: "general"}, Path: "docs/general/workflows/other.yaml", Spec: core.Spec{Text: "Independent."}}
		outputs := map[string]string{
			".codex/agents/reviewer.toml": codexOutput,
			".claude/agents/reviewer.md":  claudeOutput,
		}
		return parseImpactProject(t, []*core.Resource{policy, &agent, &consumer, &unrelated}, outputs)
	}
	impact := Changes(makeProject("old-model", "old codex output", "old claude output"), makeProject("new-model", "new codex output", "new claude output"))
	assertAffected(t, impact, "general/Agent/reviewer", "general/Workflow/review")
}

func TestChangesMapsSharedRuleSourceViewsAndProviderViewToAllApplicableResources(t *testing.T) {
	makeProject := func(ruleText, companion, providerView string) *Project {
		policy := impactProjectResource("cockpit", []core.Area{
			{Name: "cockpit-general", Path: "docs/general", Rules: []core.Ref{{Name: "shared"}}},
			{Name: "customer", Path: "docs/customer"},
		})
		policy.Spec.Targets = []string{"claude"}
		rule := core.Resource{APIVersion: core.APIVersion, Kind: "Rule", Metadata: core.Metadata{Name: "shared", Namespace: "cockpit-general"}, Path: "docs/general/rules/shared.yaml", Spec: core.Spec{Text: ruleText}}
		general := core.Resource{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "general-flow", Namespace: "cockpit-general"}, Path: "docs/general/workflows/general-flow.yaml", Spec: core.Spec{Text: "General."}}
		second := core.Resource{APIVersion: core.APIVersion, Kind: "Skill", Metadata: core.Metadata{Name: "general-skill", Namespace: "cockpit-general"}, Path: "docs/general/skills/general-skill.yaml", Spec: core.Spec{Text: "General skill."}}
		private := core.Resource{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "customer-flow", Namespace: "customer"}, Path: "docs/customer/workflows/customer-flow.yaml", Spec: core.Spec{Text: "Customer."}}
		outputs := map[string]string{
			"docs/general/rules/shared.md":    companion,
			".claude/rules/general-shared.md": providerView,
		}
		return parseImpactProject(t, []*core.Resource{policy, &rule, &general, &second, &private}, outputs)
	}
	impact := Changes(makeProject("Old policy.", "Old rule view.", "Old provider rule."), makeProject("New policy.", "New rule view.", "New provider rule."))
	assertAffected(t, impact, "cockpit-general/Rule/shared", "cockpit-general/Workflow/general-flow", "cockpit-general/Skill/general-skill")
}

func TestChangesIncludesRemovedResourceFromOldGraph(t *testing.T) {
	policy := impactProjectResource("generic", []core.Area{{Name: "general", Path: "docs/general"}})
	skill := core.Resource{APIVersion: core.APIVersion, Kind: "Skill", Metadata: core.Metadata{Name: "retired", Namespace: "general"}, Path: "docs/general/skills/retired.yaml", Spec: core.Spec{Text: "Retired."}}
	consumer := core.Resource{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "consumer", Namespace: "general"}, Path: "docs/general/workflows/consumer.yaml", Spec: core.Spec{Text: "Consumer."}}
	before := parseImpactProject(t, []*core.Resource{policy, &skill, &consumer}, nil)
	afterPolicy := impactProjectResource("generic", []core.Area{{Name: "general", Path: "docs/general"}})
	afterConsumer := consumer
	after := parseImpactProject(t, []*core.Resource{afterPolicy, &afterConsumer}, nil)
	impact := Changes(before, after)
	if !contains(impact.Affected, skill.Key()) {
		t.Fatalf("impact omitted resource removed from the old graph %s: %v", skill.Key(), impact.Affected)
	}
}

func impactProjectResource(profile string, areas []core.Area) *core.Resource {
	return &core.Resource{APIVersion: core.APIVersion, Kind: "Project", Metadata: core.Metadata{Name: "impact-test"}, Path: projectPath, Spec: core.Spec{Profile: profile, Areas: areas}}
}

func parseImpactProject(t *testing.T, resources []*core.Resource, extraFiles map[string]string) *Project {
	t.Helper()
	snapshot := &source.Snapshot{Revision: "impact-test", Files: map[string][]byte{}, Modes: map[string]string{}}
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
