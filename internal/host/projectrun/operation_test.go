package projectrun

import (
	"reflect"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

func TestNormalizeOperationDefaultsAndRejectsUnknownValues(t *testing.T) {
	if got, err := NormalizeOperation(""); err != nil || got != OperationApply {
		t.Fatalf("empty operation = %q, %v; want apply", got, err)
	}
	for _, operation := range []string{OperationApply, OperationCleanup, OperationReconcile} {
		got, err := NormalizeOperation(operation)
		if err != nil || got != operation {
			t.Errorf("NormalizeOperation(%q) = %q, %v", operation, got, err)
		}
	}
	if _, err := NormalizeOperation("verify"); err == nil {
		t.Fatal("unsupported operation was accepted")
	}
}

func TestCleanupAndReconcilePlanEveryManagerAndCheck(t *testing.T) {
	root := "root"
	left := "left"
	right := "right"
	report := projectmodel.Report{
		Managers: []projectmodel.Manager{{ID: root}, {ID: left, Parent: root}, {ID: right, Parent: root}},
		Checks: []projectmodel.Check{
			{ID: "left-check", Owner: left, Command: []string{"go", "test", "./left"}},
			{ID: "right-check", Owner: right, Command: []string{"go", "test", "./right"}},
		},
	}
	for _, operation := range []string{OperationCleanup, OperationReconcile} {
		t.Run(operation, func(t *testing.T) {
			tasks, selected, _, err := planManagers(report, map[string][]byte{}, PlanRequest{Operation: operation, Goal: "bounded operation"}, nil, nil, Limits{MaxDepth: 4})
			if err != nil {
				t.Fatalf("plan managers: %v", err)
			}
			if len(tasks) != len(report.Managers) || !selected[root] || !selected[left] || !selected[right] {
				t.Fatalf("operation did not select the full Manager tree: tasks=%+v selected=%v", tasks, selected)
			}
			checks, findings := planChecks(report, selected)
			if len(findings) != 0 || len(checks) != len(report.Checks) {
				t.Fatalf("operation checks = %+v, findings=%v; want all checks", checks, findings)
			}
			for _, check := range checks {
				if !check.Required {
					t.Errorf("full-tree operation check %s is not required", check.ID)
				}
			}
		})
	}
}

func TestCleanupAndReconcileRejectNarrowingInputs(t *testing.T) {
	for _, operation := range []string{OperationCleanup, OperationReconcile} {
		cases := []struct {
			name    string
			request PlanRequest
			impact  *projectmodel.ChangeImpact
		}{
			{name: "explicit Manager", request: PlanRequest{Operation: operation, Managers: []string{"orders"}}},
			{name: "since revision", request: PlanRequest{Operation: operation, SinceRevision: "base"}},
			{name: "change impact", request: PlanRequest{Operation: operation}, impact: &projectmodel.ChangeImpact{}},
			{name: "model edit", request: PlanRequest{Operation: operation, ModelEdit: &Mutation{}}},
		}
		for _, tc := range cases {
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				_, _, _, err := planManagers(projectmodel.Report{Managers: []projectmodel.Manager{{ID: "root"}}}, nil, tc.request, nil, tc.impact, Limits{MaxDepth: 4})
				if err == nil {
					t.Fatal("narrowing input was accepted")
				}
			})
		}
	}
}

func TestPlanDigestBindsOperation(t *testing.T) {
	plan := PlanRecord{APIVersion: APIVersion, ID: "operation-digest", Operation: OperationApply, Strictness: map[string]StrictnessProfile{}}
	apply, err := planDigest(plan)
	if err != nil {
		t.Fatal(err)
	}
	plan.Operation = OperationCleanup
	cleanup, err := planDigest(plan)
	if err != nil {
		t.Fatal(err)
	}
	if apply == cleanup {
		t.Fatal("plan digest did not change when operation changed")
	}
	plan.Strictness["orders"] = StrictnessProfile{Evidence: []string{"transaction boundary"}, Counterexamples: 1}
	strict, err := planDigest(plan)
	if err != nil {
		t.Fatal(err)
	}
	if cleanup == strict {
		t.Fatal("plan digest did not change when effective strictness changed")
	}
	plan.BriefingDigests = map[string]string{"orders": "sha256:briefing"}
	briefed, err := planDigest(plan)
	if err != nil {
		t.Fatal(err)
	}
	if strict == briefed {
		t.Fatal("plan digest did not change when briefing bindings changed")
	}
}

func TestResolveStrictnessAddsManagerRequirementsWithoutWeakeningDefault(t *testing.T) {
	runtime := Runtime{Strictness: &StrictnessConfig{
		Default:  StrictnessProfile{Evidence: []string{"test results", "public contract review"}, Counterexamples: 2},
		Managers: map[string]StrictnessProfile{"orders": {Evidence: []string{"test results", "transaction boundary"}, Counterexamples: 1}},
	}}
	got, err := ResolveStrictness(runtime, "orders")
	if err != nil {
		t.Fatal(err)
	}
	want := StrictnessProfile{Evidence: []string{"public contract review", "test results", "transaction boundary"}, Counterexamples: 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolved strictness = %+v, want %+v", got, want)
	}
	other, err := ResolveStrictness(runtime, "inventory")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(other, StrictnessProfile{Evidence: []string{"public contract review", "test results"}, Counterexamples: 2}) {
		t.Fatalf("unconfigured Manager weakened the project default: %+v", other)
	}
}

func TestValidateStrictnessRejectsLocalWeakeningAndMalformedProfiles(t *testing.T) {
	for name, config := range map[string]*StrictnessConfig{
		"negative counterexamples": {Default: StrictnessProfile{Counterexamples: -1}},
		"blank evidence":           {Managers: map[string]StrictnessProfile{"orders": {Evidence: []string{" "}}}},
		"blank Manager":            {Managers: map[string]StrictnessProfile{" ": {Evidence: []string{"evidence"}}}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateStrictness(config); err == nil {
				t.Fatal("invalid strictness config was accepted")
			}
		})
	}
}
