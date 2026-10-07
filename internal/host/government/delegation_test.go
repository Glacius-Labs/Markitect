package government

import (
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func testDelegationLimits() DelegationLimits {
	return DelegationLimits{MaxDepth: 8, MaxFanout: 8, MaxCalls: 64}
}

func TestDelegationPlanKeepsLocalWritersAndBindsDescendantContext(t *testing.T) {
	m := Compile(testSource())
	plan := BuildPlan(m, testOrder(m), realReport(t, false))
	d := BuildDelegationPlan(m, plan, testDelegationLimits())
	if d.Status != "planned" || len(d.Findings) != 0 {
		t.Fatalf("delegation plan: %+v", d)
	}
	if d.Root.Area.Key() != m.Root.Key() || len(d.Root.Work.Paths) != 0 {
		t.Fatalf("structural root received child writer paths: %+v", d.Root)
	}
	if !reflect.DeepEqual(d.Root.Paths, []string{"handler.go", "handler_test.go"}) {
		t.Fatalf("root did not receive descendant paths as read context: %+v", d.Root.Paths)
	}
	if len(d.Root.Children) != 1 {
		t.Fatalf("root children: %+v", d.Root.Children)
	}
	child := d.Root.Children[0]
	if child.Area.Name != "orders" || !reflect.DeepEqual(child.Work.Paths, []string{"handler.go", "handler_test.go"}) {
		t.Fatalf("wrong local writer assignment: %+v", child)
	}
	if !reflect.DeepEqual(d.Root.Subjects, child.Subjects) || !reflect.DeepEqual(d.Root.Paths, child.Paths) {
		t.Fatalf("parent did not bind descendant context: root=%+v child=%+v", d.Root, child)
	}
	if d.EstimatedCalls != 3 { // executor for the file owner plus one reviewer per Area
		t.Fatalf("estimated calls = %d, want 3", d.EstimatedCalls)
	}
	if BuildDelegationPlan(m, plan, testDelegationLimits()).Digest != d.Digest {
		t.Fatal("fixed delegation inputs changed digest")
	}
}

func TestDelegationTaskActionsDoNotCopyBroaderPriorMandate(t *testing.T) {
	s := testSource()
	mutate(&s, "Mandate", "orders", func(d *core.Definition) {
		d.Spec["actions"] = []any{"implement", "review", "amend-model"}
	})
	m := Compile(s)
	if len(m.Findings) != 0 {
		t.Fatalf("test model: %+v", m.Findings)
	}
	plan := BuildPlan(m, testOrder(m), realReport(t, false))
	d := BuildDelegationPlan(m, plan, testDelegationLimits())
	if d.Status != "planned" || len(d.Root.Children) != 1 {
		t.Fatalf("delegation plan: %+v", d)
	}
	local := d.Root.Children[0]
	if !reflect.DeepEqual(local.Actions, []string{"implement"}) {
		t.Fatalf("task actions copied prior mandate powers: %v", local.Actions)
	}
	if len(local.Mandates) != 1 || local.Mandates[0].Name != "orders" {
		t.Fatalf("prior Mandate identity was lost: %v", local.Mandates)
	}
	if len(d.Root.Actions) != 0 {
		t.Fatalf("structural parent received an executable action: %v", d.Root.Actions)
	}
}

func TestDelegationPlanRequiresFiniteExplicitLimitsAndBoundsCalls(t *testing.T) {
	m := Compile(testSource())
	plan := BuildPlan(m, testOrder(m), realReport(t, false))
	cases := []struct {
		name   string
		limits DelegationLimits
		code   string
	}{
		{"missing depth", DelegationLimits{MaxFanout: 8, MaxCalls: 64}, "delegation.limit.depth"},
		{"unbounded depth", DelegationLimits{MaxDepth: 33, MaxFanout: 8, MaxCalls: 64}, "delegation.limit.depth"},
		{"calls", DelegationLimits{MaxDepth: 8, MaxFanout: 8, MaxCalls: 2}, "delegation.calls"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := BuildDelegationPlan(m, plan, tc.limits)
			if d.Status != "blocked" || !hasFinding(d.Findings, tc.code) {
				t.Fatalf("wanted %s: %+v", tc.code, d)
			}
			if tc.code == "delegation.calls" && d.Root.Area.Name != "" {
				t.Fatal("call overflow was detected only after tree construction")
			}
		})
	}
}

func TestDelegationPlanRecursesByModelDepthAndRejectsDepthOverflow(t *testing.T) {
	m := Compile(testSource())
	plan := BuildPlan(m, testOrder(m), realReport(t, false))
	plan.Work = append(plan.Work, Work{Area: testID("Area", "cancellation")})
	limits := testDelegationLimits()
	d := BuildDelegationPlan(m, plan, limits)
	if d.Status != "planned" || len(d.Root.Children) != 1 || len(d.Root.Children[0].Children) != 1 {
		t.Fatalf("three-level hierarchy was not retained: %+v", d)
	}
	if got := d.Root.Children[0].Children[0].Depth; got != 3 {
		t.Fatalf("leaf depth = %d, want 3", got)
	}
	limits.MaxDepth = 2
	bounded := BuildDelegationPlan(m, plan, limits)
	if bounded.Status != "blocked" || !hasFinding(bounded.Findings, "delegation.depth") {
		t.Fatalf("depth overflow passed: %+v", bounded)
	}
	if bounded.Root.Area.Name != "" {
		t.Fatal("depth overflow was detected only after recursive tree construction")
	}
}

func TestDelegationPlanRejectsWriterCollisionAcrossAreas(t *testing.T) {
	m := Compile(testSource())
	plan := BuildPlan(m, testOrder(m), realReport(t, false))
	plan.Work = append(plan.Work, Work{
		Area:  testID("Area", "root"),
		Paths: []string{"handler.go"},
	})
	// Keep the synthetic root assignment well-formed enough to exercise the
	// candidate-local exclusive writer check.
	plan.Work[len(plan.Work)-1].Subjects = []core.DefinitionIdentity{testID("Rule", "cancel")}
	d := BuildDelegationPlan(m, plan, testDelegationLimits())
	if d.Status != "blocked" || !hasFinding(d.Findings, "delegation.writer.collision") {
		t.Fatalf("writer collision passed: %+v", d)
	}
}

func TestDelegationPlanRevalidatesLocalAuthorityAndSubjectScope(t *testing.T) {
	m := Compile(testSource())
	plan := BuildPlan(m, testOrder(m), realReport(t, false))
	plan.Work[0].Mandates = nil
	d := BuildDelegationPlan(m, plan, testDelegationLimits())
	if d.Status != "blocked" || !hasFinding(d.Findings, "delegation.authority") {
		t.Fatalf("missing local mandate passed: %+v", d)
	}

	plan = BuildPlan(m, testOrder(m), realReport(t, false))
	plan.Work[0].Subjects = nil
	d = BuildDelegationPlan(m, plan, testDelegationLimits())
	if d.Status != "blocked" || !hasFinding(d.Findings, "delegation.scope.missing") {
		t.Fatalf("omitted path realization scope passed: %+v", d)
	}
}

func TestDelegationPlanRejectsDuplicateEmptyWorkEntries(t *testing.T) {
	m := Compile(testSource())
	plan := BuildPlan(m, testOrder(m), realReport(t, false))
	plan.Work = append(plan.Work, Work{Area: m.Root}, Work{Area: m.Root})
	d := BuildDelegationPlan(m, plan, testDelegationLimits())
	if d.Status != "blocked" || !hasFinding(d.Findings, "delegation.work.duplicate") {
		t.Fatalf("duplicate empty Work entries passed: %+v", d)
	}
}
