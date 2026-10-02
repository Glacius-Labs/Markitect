package core

import "testing"

func TestProviderAdapterMappingsAndRetirementsAreValidated(t *testing.T) {
	p := project(Area{Name: "sample", Path: "docs"})
	p.Spec.ProviderAdapters = &ProviderAdapters{RuleSources: map[string][]string{"empty": {}}, RetiredSkills: []string{"../escape", "active", "old", "old"}, RetiredAgents: []string{"active-agent"}}
	s := resource("Skill", "sample", "active", "docs/active.yaml")
	a := resource("Agent", "sample", "active-agent", "docs/active-agent.yaml")
	g := Build([]*Resource{p, s, a})
	for _, code := range []string{"provider-adapter.rule", "provider-adapter.retired-name", "provider-adapter.retired-duplicate", "provider-adapter.retired-active"} {
		if !hasCode(g, code) {
			t.Errorf("missing %s: %#v", code, g.Diagnostics)
		}
	}
}

func resource(kind, namespace, name, file string) *Resource {
	return &Resource{APIVersion: APIVersion, Kind: kind, Metadata: Metadata{Name: name, Namespace: namespace}, Path: file}
}

func project(areas ...Area) *Resource {
	r := resource("Project", "", "test", "markitect.yaml")
	r.Spec.Areas = areas
	return r
}

func hasCode(g *Graph, code string) bool {
	for _, d := range g.Diagnostics {
		if d.Code == code {
			return true
		}
	}
	return false
}

func hasEdge(g *Graph, from, to string) bool {
	for _, edge := range g.Edges[from] {
		if edge == to {
			return true
		}
	}
	return false
}

func TestBuildResolvesTypedUsesAndRejectsMissingWrongKindAndScope(t *testing.T) {
	p := project(Area{Name: "general", Path: "docs/general"}, Area{Name: "area-b", Path: "docs/area-b"})
	s := resource("Skill", "general", "entry", "docs/general/entry.yaml")
	s.Spec.Uses = []Ref{
		{Kind: "Workflow", Name: "missing"},
		{Kind: "Rule", Name: "policy"},
		{Kind: "Text", Name: "private-note", Namespace: "area-b"},
	}
	rule := resource("Rule", "general", "policy", "docs/general/policy.yaml")
	text := resource("Text", "area-b", "private-note", "docs/area-b/private.yaml")
	g := Build([]*Resource{p, s, rule, text})
	for _, code := range []string{"reference.missing", "reference.kind", "reference.scope"} {
		if !hasCode(g, code) {
			t.Errorf("expected %s diagnostic; got %#v", code, g.Diagnostics)
		}
	}
}

func TestBuildUsesLongestAreaAndInheritsAncestorRules(t *testing.T) {
	p := project(
		Area{Name: "general", Path: "docs/general", Rules: []Ref{{Name: "documentation"}}},
		Area{Name: "product-b", Path: "docs/general/projects/product-b", Imports: []string{"general"}},
	)
	rule := resource("Rule", "general", "documentation", "docs/general/documentation.yaml")
	w := resource("Workflow", "product-b", "author", "docs/general/projects/product-b/author.yaml")
	spoof := resource("Text", "general", "local", "docs/general/projects/product-b/local.yaml")
	g := Build([]*Resource{w, spoof, p, rule})
	if !hasEdge(g, w.Key(), rule.Key()) {
		t.Errorf("ancestor rule was not inherited: %#v", g.Edges[w.Key()])
	}
	if !hasCode(g, "area.namespace") {
		t.Errorf("expected namespace spoof diagnostic; got %#v", g.Diagnostics)
	}
}

func TestAreaNamesRejectPathTraversalWithoutRestrictingOrdinaryCharacters(t *testing.T) {
	for _, name := range []string{"text-v0", "area 1"} {
		project := resource("Project", "", "sample", "markitect.yaml")
		project.Spec.Areas = []Area{{Name: name, Path: "docs/area"}}
		graph := Build([]*Resource{project})
		for _, diagnostic := range graph.Diagnostics {
			if diagnostic.Code == "area.invalid" {
				t.Errorf("valid Area name %q was rejected: %+v", name, diagnostic)
			}
		}
	}
	for _, name := range []string{"..", "bad/name", "bad\\name", "bad:name", "bad\x00name"} {
		project := resource("Project", "", "sample", "markitect.yaml")
		project.Spec.Areas = []Area{{Name: name, Path: "docs/area"}}
		graph := Build([]*Resource{project})
		found := false
		for _, diagnostic := range graph.Diagnostics {
			found = found || diagnostic.Code == "area.invalid"
		}
		if !found {
			t.Errorf("unsafe Area name %q was accepted", name)
		}
	}
}

func TestBuildResolvesContractBindingAndChecksExactSignature(t *testing.T) {
	p := project(Area{Name: "general", Path: "docs/general"})
	c := resource("Contract", "general", "review", "docs/general/review.yaml")
	c.Spec.Kind, c.Spec.Input, c.Spec.Output = "Agent", []string{"change"}, []string{"findings"}
	a := resource("Agent", "general", "reviewer", "docs/general/reviewer.yaml")
	a.Spec.Input, a.Spec.Output = []string{"change"}, []string{"findings"}
	a.Spec.Implements = []Ref{{Name: "review"}}
	w := resource("Workflow", "general", "review-workflow", "docs/general/workflow.yaml")
	w.Spec.Needs = []Ref{{Name: "review"}}
	p.Spec.Bindings = []Binding{{Contract: Ref{Kind: "Contract", Name: "review", Namespace: "general"}, Implementation: Ref{Kind: "Agent", Name: "reviewer", Namespace: "general"}}}
	g := Build([]*Resource{p, c, a, w})
	if len(g.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", g.Diagnostics)
	}
	if !hasEdge(g, w.Key(), a.Key()) || !hasEdge(g, w.Key(), c.Key()) || !hasEdge(g, a.Key(), c.Key()) {
		t.Fatalf("missing context/binding edges: %#v", g.Edges)
	}
	// A context edge from the implementation to its contract is not an
	// execution edge and must not create a cycle by itself.
	if hasCode(g, "graph.cycle") {
		t.Fatalf("contract context edge created a false cycle: %#v", g.Diagnostics)
	}
	a.Spec.Output = []string{"review"}
	if bad := Build([]*Resource{p, c, a, w}); !hasCode(bad, "contract.signature") {
		t.Fatalf("expected exact-signature diagnostic: %#v", bad.Diagnostics)
	}
}

func TestBuildVerifiesUnboundImplementationsAndAllowsCompatibleAlternatives(t *testing.T) {
	p := project(Area{Name: "general", Path: "docs/general"})
	c := resource("Contract", "general", "review", "docs/general/review.yaml")
	c.Spec.Kind, c.Spec.Input, c.Spec.Output = "Agent", []string{"change"}, []string{"findings"}
	selected := resource("Agent", "general", "reviewer-a", "docs/general/reviewer-a.yaml")
	selected.Spec.Input, selected.Spec.Output = []string{"change"}, []string{"findings"}
	selected.Spec.Implements = []Ref{{Name: "review"}}
	alternative := resource("Agent", "general", "reviewer-b", "docs/general/reviewer-b.yaml")
	alternative.Spec.Input, alternative.Spec.Output = []string{"change"}, []string{"summary"}
	alternative.Spec.Implements = []Ref{{Name: "review"}}
	consumer := resource("Workflow", "general", "review-workflow", "docs/general/workflow.yaml")
	consumer.Spec.Needs = []Ref{{Name: "review"}}
	p.Spec.Bindings = []Binding{{Contract: Ref{Kind: "Contract", Name: "review", Namespace: "general"}, Implementation: Ref{Kind: "Agent", Name: "reviewer-a", Namespace: "general"}}}

	invalid := Build([]*Resource{p, c, selected, alternative, consumer})
	if !hasCode(invalid, "contract.signature") {
		t.Fatalf("unbound incompatible implementation was not diagnosed: %#v", invalid.Diagnostics)
	}

	alternative.Spec.Output = []string{"findings"}
	valid := Build([]*Resource{p, c, selected, alternative, consumer})
	if len(valid.Diagnostics) != 0 {
		t.Fatalf("compatible alternative implementation should be valid: %#v", valid.Diagnostics)
	}
	if !hasEdge(valid, consumer.Key(), selected.Key()) || hasEdge(valid, consumer.Key(), alternative.Key()) {
		t.Fatalf("consumer did not use only its selected implementation: %#v", valid.Edges[consumer.Key()])
	}
}

func TestBuildReportsUnboundNeedsAndUnsupportedTarget(t *testing.T) {
	p := project(Area{Name: "general", Path: "docs/general"})
	p.Spec.Targets = []string{"codex", "unknown"}
	c := resource("Contract", "general", "review", "docs/general/review.yaml")
	c.Spec.Kind = "Agent"
	w := resource("Workflow", "general", "run", "docs/general/run.yaml")
	w.Spec.Needs = []Ref{{Name: "review"}}
	g := Build([]*Resource{p, c, w})
	for _, code := range []string{"project.target", "binding.missing"} {
		if !hasCode(g, code) {
			t.Errorf("expected %s diagnostic; got %#v", code, g.Diagnostics)
		}
	}
}

func TestBuildValidatesProjectChecks(t *testing.T) {
	tests := []struct {
		name   string
		checks []Check
		want   string
	}{
		{name: "valid argv", checks: []Check{{Name: "unit-test", Run: []string{"go", "test", "./..."}}}},
		{name: "duplicate names", checks: []Check{{Name: "verify", Run: []string{"tool"}}, {Name: "verify", Run: []string{"other"}}}, want: "check.duplicate"},
		{name: "invalid name", checks: []Check{{Name: "../verify", Run: []string{"tool"}}}, want: "check.invalid"},
		{name: "missing argv", checks: []Check{{Name: "verify"}}, want: "check.invalid"},
		{name: "blank executable", checks: []Check{{Name: "verify", Run: []string{" \t", "arg"}}}, want: "check.invalid"},
		{name: "path executable", checks: []Check{{Name: "verify", Run: []string{"./scripts/verify"}}}, want: "check.invalid"},
		{name: "NUL in argument", checks: []Check{{Name: "verify", Run: []string{"tool", "bad\x00arg"}}}, want: "check.invalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := project(Area{Name: "general", Path: "docs/general"})
			p.Spec.Checks = tt.checks
			g := Build([]*Resource{p})
			if tt.want == "" {
				if len(g.Diagnostics) != 0 {
					t.Fatalf("valid project checks produced diagnostics: %#v", g.Diagnostics)
				}
				if len(g.Project.Spec.Checks) != len(tt.checks) || g.Project.Spec.Checks[0].Run[1] != "test" {
					t.Fatalf("check argv/order changed: %#v", g.Project.Spec.Checks)
				}
				return
			}
			if !hasCode(g, tt.want) {
				t.Fatalf("expected %s diagnostic, got %#v", tt.want, g.Diagnostics)
			}
		})
	}
}

func TestBuildRejectsChecksOnNonProjectResources(t *testing.T) {
	p := project(Area{Name: "general", Path: "docs/general"})
	r := resource("Rule", "general", "policy", "docs/general/policy.yaml")
	r.Spec.Text = "Policy."
	r.Spec.Checks = []Check{{Name: "validate", Run: []string{"validator"}}}
	if g := Build([]*Resource{p, r}); !hasCode(g, "check.kind") {
		t.Fatalf("expected non-Project checks diagnostic, got %#v", g.Diagnostics)
	}
}

func TestBuildResolvesProjectRuleAdapterSourcesAcrossAreas(t *testing.T) {
	p := project(Area{Name: "general", Path: "docs/general"}, Area{Name: "platform", Path: "docs/platform"})
	text := resource("Text", "platform", "adapter-source", "docs/platform/adapter.yaml")
	p.Spec.RuleAdapters = map[string][]Ref{"agents": {{Kind: "Text", Name: "adapter-source", Namespace: "platform"}}}
	g := Build([]*Resource{p, text})
	if len(g.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", g.Diagnostics)
	}
	if !hasEdge(g, p.Key(), text.Key()) {
		t.Fatalf("project adapter source edge missing: %#v", g.Edges[p.Key()])
	}
}

func TestBuildDetectsRuntimeCyclesAndDuplicateProviderNames(t *testing.T) {
	p := project(Area{Name: "general", Path: "docs/general"}, Area{Name: "other", Path: "docs/other"})
	w1 := resource("Workflow", "general", "prepare", "docs/general/prepare.yaml")
	w2 := resource("Workflow", "general", "publish", "docs/general/publish.yaml")
	w1.Spec.Uses = []Ref{{Kind: "Workflow", Name: "publish"}}
	w2.Spec.Uses = []Ref{{Kind: "Workflow", Name: "prepare"}}
	s1 := resource("Skill", "general", "start", "docs/general/start.yaml")
	s2 := resource("Skill", "other", "start", "docs/other/start.yaml")
	g := Build([]*Resource{p, w1, w2, s1, s2})
	if !hasCode(g, "graph.cycle") {
		t.Errorf("expected runtime cycle diagnostic; got %#v", g.Diagnostics)
	}
	if !hasCode(g, "provider.name") {
		t.Errorf("expected provider name collision; got %#v", g.Diagnostics)
	}
}

func TestBuildDiagnosticsAreDeterministic(t *testing.T) {
	p := project(Area{Name: "general", Path: "docs/general"})
	a := resource("Skill", "general", "a", "docs/general/a.yaml")
	a.Spec.Uses = []Ref{{Kind: "Workflow", Name: "missing-z"}, {Kind: "Workflow", Name: "missing-a"}}
	b := resource("Agent", "general", "b", "docs/general/b.yaml")
	b.Spec.Uses = []Ref{{Kind: "Skill", Name: "missing"}}
	first, second := Build([]*Resource{p, a, b}), Build([]*Resource{b, a, p})
	if len(first.Diagnostics) != len(second.Diagnostics) {
		t.Fatalf("diagnostic counts differ")
	}
	for i := range first.Diagnostics {
		if first.Diagnostics[i] != second.Diagnostics[i] {
			t.Fatalf("diagnostic order differs: %#v vs %#v", first.Diagnostics, second.Diagnostics)
		}
	}
}
