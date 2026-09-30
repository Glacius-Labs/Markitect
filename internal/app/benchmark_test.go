package app

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/source"
)

// TestAuthoringScenarioImpactOracle keeps a synthetic multi-area graph as an
// exact-set oracle. Each input is a fixed in-memory snapshot.
func TestAuthoringScenarioImpactOracle(t *testing.T) {
	tests := []struct {
		name          string
		before, after func(testing.TB) *Project
		want          []string
	}{
		{"local Rule", func(tb testing.TB) *Project { return authoringProject(tb, "base", authoringChange{}) }, func(tb testing.TB) *Project {
			return authoringProject(tb, "candidate", authoringChange{localRule: true})
		}, []string{"alpha/Rule/local-policy", "alpha/Skill/rollback-review", "alpha/Workflow/rollback-flow"}},
		{"shared Rule", func(tb testing.TB) *Project { return authoringProject(tb, "base", authoringChange{}) }, func(tb testing.TB) *Project {
			return authoringProject(tb, "candidate", authoringChange{sharedRule: true})
		}, []string{"alpha/Agent/alternate-reviewer", "alpha/Agent/rollback-reviewer", "alpha/Contract/rollback-review", "alpha/Rule/local-policy", "alpha/Skill/rollback-review", "alpha/Workflow/rollback-flow", "beta/Rule/local-policy", "beta/Skill/beta-review", "beta/Workflow/beta-flow", "shared/Rule/shared-policy"}},
		{"removed dependency keeps old closure", func(tb testing.TB) *Project { return authoringProject(tb, "base", authoringChange{}) }, func(tb testing.TB) *Project {
			return authoringProject(tb, "candidate", authoringChange{removeSkillRule: true})
		}, []string{"alpha/Skill/rollback-review", "alpha/Workflow/rollback-flow"}},
		{"selected binding change invalidates the configured project", func(tb testing.TB) *Project { return authoringProject(tb, "base", authoringChange{}) }, func(tb testing.TB) *Project {
			return authoringProject(tb, "candidate", authoringChange{bindingChange: true})
		}, []string{"/Project/sample", "alpha/Agent/alternate-reviewer", "alpha/Agent/rollback-reviewer", "alpha/Contract/rollback-review", "alpha/Rule/local-policy", "alpha/Skill/rollback-review", "alpha/Workflow/rollback-flow", "beta/Rule/local-policy", "beta/Skill/beta-review", "beta/Workflow/beta-flow", "shared/Rule/shared-policy"}},
		{"unknown file is conservative", func(tb testing.TB) *Project { return authoringProject(tb, "base", authoringChange{}) }, func(tb testing.TB) *Project {
			return authoringProject(tb, "candidate", authoringChange{unknownFile: true})
		}, []string{"/Project/sample", "alpha/Agent/alternate-reviewer", "alpha/Agent/rollback-reviewer", "alpha/Contract/rollback-review", "alpha/Rule/local-policy", "alpha/Skill/rollback-review", "alpha/Workflow/rollback-flow", "beta/Rule/local-policy", "beta/Skill/beta-review", "beta/Workflow/beta-flow", "shared/Rule/shared-policy"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Changes(tt.before(t), tt.after(t)).Affected
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("affected = %v, want exact oracle %v", got, tt.want)
			}
		})
	}
}

func TestAuthoringScenarioContextSelectsBoundImplementation(t *testing.T) {
	p := authoringProject(t, "fixed-review-snapshot", authoringChange{})
	ctx, err := CompileContext(p, "alpha/Workflow/rollback-flow", "authoring-scenario")
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(ctx.Inputs))
	for _, input := range ctx.Inputs {
		got = append(got, input.Key)
	}
	want := []string{"/Project/sample", "alpha/Agent/rollback-reviewer", "alpha/Contract/rollback-review", "alpha/Rule/local-policy", "alpha/Skill/rollback-review", "alpha/Workflow/rollback-flow", "shared/Rule/shared-policy"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("compiled context entries = %v, want %v", got, want)
	}
	if ctx.SnapshotDigest != p.Snapshot.Digest() || ctx.Provisional {
		t.Fatalf("context is not bound to the fixed snapshot: %#v", ctx)
	}
}

// BenchmarkAuthoringContext measures context compilation only; parsing and
// fixture correctness checks are outside the timed section.
func BenchmarkAuthoringContext(b *testing.B) {
	p := authoringBenchmarkProject(b)
	entry := "area-00/Workflow/flow-00"
	probe, err := CompileContext(p, entry, "benchmark")
	if err != nil {
		b.Fatalf("compile benchmark context fixture: %v", err)
	}
	if len(probe.Inputs) < 8 {
		b.Fatalf("invalid fixture: inputs=%d", len(probe.Inputs))
	}
	b.ResetTimer()
	b.ReportAllocs()
	var last *Context
	for i := 0; i < b.N; i++ {
		last, err = CompileContext(p, entry, "benchmark")
		if err != nil {
			b.Fatalf("compile context: %v", err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(contextSize(b, probe)), "context-bytes/op")
	b.ReportMetric(float64(len(probe.Inputs)), "context-entries/op")
	if len(last.Inputs) != len(probe.Inputs) || last.Digest != probe.Digest {
		b.Fatal("context diverged from checked fixture")
	}
}

// BenchmarkAuthoringImpact measures old/new closure calculation only; snapshots
// are parsed and the expected result is checked before timing begins.
func BenchmarkAuthoringImpact(b *testing.B) {
	before := authoringBenchmarkProject(b)
	after, err := Parse(authoringBenchmarkSnapshot(b, "benchmark-candidate", true))
	if err != nil {
		b.Fatalf("parse benchmark candidate: %v", err)
	}
	if len(after.Diagnostics) != 0 {
		b.Fatalf("invalid candidate: diagnostics=%v", after.Diagnostics)
	}
	probe := Changes(before, after)
	if len(probe.Affected) < 10 || !contains(probe.Affected, "area-00/Rule/shared-policy") || !contains(probe.Affected, "area-09/Workflow/flow-11") {
		b.Fatalf("invalid impact fixture: affected=%d", len(probe.Affected))
	}
	b.ResetTimer()
	b.ReportAllocs()
	var last *Impact
	for i := 0; i < b.N; i++ {
		last = Changes(before, after)
	}
	b.StopTimer()
	b.ReportMetric(float64(len(probe.Affected)), "affected-entries/op")
	if !reflect.DeepEqual(last.Affected, probe.Affected) || !reflect.DeepEqual(last.Changed, probe.Changed) {
		b.Fatal("impact diverged from checked fixture")
	}
}

type authoringChange struct{ localRule, sharedRule, removeSkillRule, bindingChange, unknownFile bool }

func authoringProject(tb testing.TB, revision string, change authoringChange) *Project {
	tb.Helper()
	project := core.Resource{APIVersion: core.APIVersion, Kind: "Project", Metadata: core.Metadata{Name: "sample"}, Path: "markitect.yaml", Spec: core.Spec{
		Profile:  "generic",
		Areas:    []core.Area{{Name: "shared", Path: "docs/shared"}, {Name: "alpha", Path: "docs/alpha", Imports: []string{"shared"}, Rules: []core.Ref{{Kind: "Rule", Name: "shared-policy", Namespace: "shared"}}}, {Name: "beta", Path: "docs/beta", Imports: []string{"shared"}, Rules: []core.Ref{{Kind: "Rule", Name: "shared-policy", Namespace: "shared"}}}},
		Bindings: []core.Binding{{Contract: core.Ref{Kind: "Contract", Name: "rollback-review", Namespace: "alpha"}, Implementation: core.Ref{Kind: "Agent", Name: "rollback-reviewer", Namespace: "alpha"}}},
	}}
	sharedText, localText := "Require rollback evidence.", "Keep rollback scoped."
	if change.sharedRule {
		sharedText += " Revised."
	}
	if change.localRule {
		localText += " Revised."
	}
	if change.bindingChange {
		project.Spec.Bindings[0].Implementation = core.Ref{Kind: "Agent", Name: "alternate-reviewer", Namespace: "alpha"}
	}
	resources := []*core.Resource{
		&project,
		{APIVersion: core.APIVersion, Kind: "Rule", Metadata: core.Metadata{Name: "shared-policy", Namespace: "shared"}, Path: "docs/shared/rules/shared-policy.yaml", Spec: core.Spec{Text: sharedText}},
		{APIVersion: core.APIVersion, Kind: "Rule", Metadata: core.Metadata{Name: "local-policy", Namespace: "alpha"}, Path: "docs/alpha/rules/local-policy.yaml", Spec: core.Spec{Text: localText}},
		{APIVersion: core.APIVersion, Kind: "Rule", Metadata: core.Metadata{Name: "local-policy", Namespace: "beta"}, Path: "docs/beta/rules/local-policy.yaml", Spec: core.Spec{Text: "Keep beta isolated."}},
		{APIVersion: core.APIVersion, Kind: "Contract", Metadata: core.Metadata{Name: "rollback-review", Namespace: "alpha"}, Path: "docs/alpha/contracts/rollback-review.yaml", Spec: core.Spec{Text: "Review a rollback.", Kind: "Agent", Input: []string{"change"}, Output: []string{"findings"}}},
		{APIVersion: core.APIVersion, Kind: "Agent", Metadata: core.Metadata{Name: "rollback-reviewer", Namespace: "alpha"}, Path: "docs/alpha/agents/rollback-reviewer.yaml", Spec: core.Spec{Text: "Assess rollback risk.", Implements: []core.Ref{{Kind: "Contract", Name: "rollback-review"}}, Input: []string{"change"}, Output: []string{"findings"}}},
		{APIVersion: core.APIVersion, Kind: "Agent", Metadata: core.Metadata{Name: "alternate-reviewer", Namespace: "alpha"}, Path: "docs/alpha/agents/alternate-reviewer.yaml", Spec: core.Spec{Text: "Alternative implementation.", Implements: []core.Ref{{Kind: "Contract", Name: "rollback-review"}}, Input: []string{"change"}, Output: []string{"findings"}}},
	}
	alphaSkill := core.Resource{APIVersion: core.APIVersion, Kind: "Skill", Metadata: core.Metadata{Name: "rollback-review", Namespace: "alpha"}, Path: "docs/alpha/skills/rollback-review.yaml", Spec: core.Spec{Text: "Review rollback changes."}}
	if !change.removeSkillRule {
		alphaSkill.Spec.Rules = []core.Ref{{Kind: "Rule", Name: "local-policy"}}
	}
	resources = append(resources, &alphaSkill)
	resources = append(resources,
		&core.Resource{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "rollback-flow", Namespace: "alpha"}, Path: "docs/alpha/workflows/rollback-flow.yaml", Spec: core.Spec{Text: "Run the review.", Uses: []core.Ref{{Kind: "Skill", Name: "rollback-review"}}, Needs: []core.Ref{{Kind: "Contract", Name: "rollback-review"}}}},
		&core.Resource{APIVersion: core.APIVersion, Kind: "Skill", Metadata: core.Metadata{Name: "beta-review", Namespace: "beta"}, Path: "docs/beta/skills/beta-review.yaml", Spec: core.Spec{Text: "Beta review."}},
		&core.Resource{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: "beta-flow", Namespace: "beta"}, Path: "docs/beta/workflows/beta-flow.yaml", Spec: core.Spec{Text: "Beta flow.", Uses: []core.Ref{{Kind: "Skill", Name: "beta-review"}}}},
	)
	snapshot := &source.Snapshot{Revision: revision, Files: map[string][]byte{}, Modes: map[string]string{}}
	for _, resource := range resources {
		data, err := format.Encode(*resource)
		if err != nil {
			tb.Fatal(err)
		}
		snapshot.Files[resource.Path], snapshot.Modes[resource.Path] = data, "100644"
	}
	if change.unknownFile {
		snapshot.Files["docs/shared/notes.md"], snapshot.Modes["docs/shared/notes.md"] = []byte("An unclassified input changed.\n"), "100644"
	}
	p, err := Parse(snapshot)
	if err != nil {
		tb.Fatal(err)
	}
	if len(p.Diagnostics) != 0 {
		tb.Fatalf("invalid authoring fixture: %#v", p.Diagnostics)
	}
	return p
}

func authoringBenchmarkProject(tb testing.TB) *Project {
	tb.Helper()
	p, err := Parse(authoringBenchmarkSnapshot(tb, "benchmark-base", false))
	if err != nil {
		tb.Fatal(err)
	}
	if len(p.Diagnostics) != 0 {
		tb.Fatalf("invalid benchmark fixture: %#v", p.Diagnostics)
	}
	return p
}

func authoringBenchmarkSnapshot(tb testing.TB, revision string, revise bool) *source.Snapshot {
	tb.Helper()
	areas := make([]core.Area, 10)
	for i := range areas {
		areas[i] = core.Area{Name: fmt.Sprintf("area-%02d", i), Path: fmt.Sprintf("docs/area-%02d", i), Rules: []core.Ref{{Kind: "Rule", Name: "shared-policy", Namespace: "area-00"}}}
	}
	project := core.Resource{APIVersion: core.APIVersion, Kind: "Project", Metadata: core.Metadata{Name: "benchmark"}, Path: "markitect.yaml", Spec: core.Spec{Profile: "generic", Areas: areas}}
	resources := []*core.Resource{&project}
	sharedText := "Shared authoring constraint."
	if revise {
		sharedText += " Candidate revision."
	}
	resources = append(resources, &core.Resource{APIVersion: core.APIVersion, Kind: "Rule", Metadata: core.Metadata{Name: "shared-policy", Namespace: "area-00"}, Path: "docs/area-00/rules/shared-policy.yaml", Spec: core.Spec{Text: sharedText}})
	for area := range areas {
		ns := fmt.Sprintf("area-%02d", area)
		for item := 0; item < 12; item++ {
			name := fmt.Sprintf("note-%02d", item)
			resources = append(resources, &core.Resource{APIVersion: core.APIVersion, Kind: "Text", Metadata: core.Metadata{Name: name, Namespace: ns}, Path: fmt.Sprintf("docs/%s/text/%s.yaml", ns, name), Spec: core.Spec{Text: "Synthetic evidence input for the bounded authoring context benchmark."}})
		}
		for item := 0; item < 24; item++ {
			name := fmt.Sprintf("%s-skill-%02d", ns, item)
			uses := make([]core.Ref, 6)
			for note := range uses {
				uses[note] = core.Ref{Kind: "Text", Name: fmt.Sprintf("note-%02d", note)}
			}
			resources = append(resources, &core.Resource{APIVersion: core.APIVersion, Kind: "Skill", Metadata: core.Metadata{Name: name, Namespace: ns}, Path: fmt.Sprintf("docs/%s/skills/%s.yaml", ns, name), Spec: core.Spec{Text: "A bounded synthetic workflow skill with enough prose to exercise context selection and hashing.", Uses: uses}})
		}
		for item := 0; item < 12; item++ {
			name := fmt.Sprintf("flow-%02d", item)
			resources = append(resources, &core.Resource{APIVersion: core.APIVersion, Kind: "Workflow", Metadata: core.Metadata{Name: name, Namespace: ns}, Path: fmt.Sprintf("docs/%s/workflows/%s.yaml", ns, name), Spec: core.Spec{Text: "Synthetic review flow with explicitly declared uses and shared authoring rules.", Uses: []core.Ref{{Kind: "Skill", Name: fmt.Sprintf("%s-skill-%02d", ns, item)}}}})
		}
	}
	snapshot := &source.Snapshot{Revision: revision, Files: map[string][]byte{}, Modes: map[string]string{}}
	for _, resource := range resources {
		data, err := format.Encode(*resource)
		if err != nil {
			tb.Fatal(err)
		}
		snapshot.Files[resource.Path], snapshot.Modes[resource.Path] = data, "100644"
	}
	return snapshot
}

func contextSize(tb testing.TB, ctx *Context) int {
	tb.Helper()
	data, err := format.Encode(ctx)
	if err != nil {
		tb.Fatal(err)
	}
	return len(data)
}
