package run

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

var closedManagers = []string{".", "a"}

func closedRequest(mandate Mandate) Request {
	return Request{ID: "run-1", Project: "p", Mandate: mandate, Input: project.CandidateRef{Project: "p", Binding: "binding", Key: "input", Commit: "c0", Tree: "t0"}}
}

// closedResult is a result ValidateClosed accepts for closedRequest(mandate)
// and closedManagers; each refusal test breaks exactly one property of it.
func closedResult(mandate Mandate) Result {
	r := closedRequest(mandate)
	v := Result{ID: r.ID, Project: r.Project, Binding: "binding", Mandate: mandate, Technical: Completed, Decision: Approve, Input: r.Input}
	candidate := r.Input
	if mandate == Implement {
		v.Decision = Submitted
		candidate.Key, candidate.Commit, candidate.Tree = "output", "c1", "t1"
	}
	v.Candidate = &candidate
	for _, pkg := range closedManagers {
		v.Cleanup = append(v.Cleanup, Closure{Resource: sessionResource(pkg), Closed: true}, Closure{Resource: copyResource(pkg), Closed: true})
		v.Handoffs = append(v.Handoffs, Handoff{Manager: pkg})
	}
	v.Cleanup = append(v.Cleanup, Closure{Resource: workersResource, Closed: true}, Closure{Resource: gitServiceResource, Closed: true})
	return v
}

func TestValidateClosedAcceptsEveryMandate(t *testing.T) {
	for _, m := range []Mandate{Assessment, Implement, Verify} {
		if err := validateClosed(closedResult(m), closedRequest(m), "binding", closedManagers); err != nil {
			t.Fatalf("%s: %v", m, err)
		}
	}
	// Rejection and model blocker are valid for every mandate with a concrete Finding.
	for _, m := range []Mandate{Assessment, Implement, Verify} {
		for _, d := range []Decision{Reject, ModelBlocker} {
			v := closedResult(m)
			v.Decision, v.Findings = d, []Finding{{Manager: ".", Text: "concrete"}}
			if err := validateClosed(v, closedRequest(m), "binding", closedManagers); err != nil {
				t.Fatalf("%s %s: %v", m, d, err)
			}
		}
	}
}

func TestValidateClosedRefusesAnyBrokenClosureProperty(t *testing.T) {
	without := func(resource string) func(*Result) {
		return func(v *Result) {
			var kept []Closure
			for _, c := range v.Cleanup {
				if c.Resource != resource {
					kept = append(kept, c)
				}
			}
			v.Cleanup = kept
		}
	}
	for _, tc := range []struct {
		name    string
		mandate Mandate
		change  func(*Result)
		want    string
	}{
		{"other Run", Verify, func(v *Result) { v.ID = "run-2" }, "complete bound"},
		{"other Project", Verify, func(v *Result) { v.Project = "q" }, "complete bound"},
		{"other binding", Verify, func(v *Result) { v.Binding = "other" }, "complete bound"},
		{"other mandate", Verify, func(v *Result) { v.Mandate = Assessment }, "complete bound"},
		{"other input", Verify, func(v *Result) { v.Input.Commit = "c9" }, "complete bound"},
		{"failed technically", Verify, func(v *Result) { v.Technical = Failed }, "complete bound"},
		{"incomplete technically", Verify, func(v *Result) { v.Technical = Incomplete }, "complete bound"},
		{"recorded errors", Verify, func(v *Result) { v.Errors = []string{"close failed"} }, "complete bound"},
		{"no candidate", Verify, func(v *Result) { v.Candidate = nil }, "complete bound"},
		{"candidate of another Project", Verify, func(v *Result) { v.Candidate.Project = "q" }, "candidate binding"},
		{"candidate of another binding", Verify, func(v *Result) { v.Candidate.Binding = "other" }, "candidate binding"},
		{"candidate without key", Verify, func(v *Result) { v.Candidate.Key = "" }, "candidate binding"},
		{"candidate of another format", Verify, func(v *Result) { v.Candidate.Format = "other" }, "candidate binding"},
		{"candidate without commit", Implement, func(v *Result) { v.Candidate.Commit = "" }, "candidate binding"},
		{"candidate without tree", Implement, func(v *Result) { v.Candidate.Tree = "" }, "candidate binding"},
		{"assessment changed commit", Assessment, func(v *Result) { v.Candidate.Commit = "c1" }, "fixed input"},
		{"verify changed tree", Verify, func(v *Result) { v.Candidate.Tree = "t1" }, "fixed input"},
		{"implement submitted nothing", Implement, func(v *Result) { v.Decision = "" }, "invalid decision"},
		{"implement approved", Implement, func(v *Result) { v.Decision = Approve }, "invalid decision"},
		{"verify submitted", Verify, func(v *Result) { v.Decision = Submitted }, "invalid decision"},
		{"assessment submitted", Assessment, func(v *Result) { v.Decision = Submitted }, "invalid decision"},
		{"rejection without Findings", Verify, func(v *Result) { v.Decision = Reject }, "mandatory Findings"},
		{"blocker with blank Finding", Implement, func(v *Result) { v.Decision, v.Findings = ModelBlocker, []Finding{{Text: " \n"}} }, "mandatory Findings"},
		{"uncertain closure", Verify, func(v *Result) { v.Cleanup[0].Closed = false }, "uncertain"},
		{"closure with error", Verify, func(v *Result) { v.Cleanup[0].Error = "kill refused" }, "uncertain"},
		{"duplicated closure", Verify, func(v *Result) { v.Cleanup = append(v.Cleanup, v.Cleanup[0]) }, "duplicated"},
		{"missing session closure", Verify, without(sessionResource("a")), "Manager resources"},
		{"missing copy closure", Verify, without(copyResource(".")), "Manager resources"},
		{"missing workers closure", Verify, without(workersResource), "hierarchy or Git"},
		{"missing Git service closure", Verify, without(gitServiceResource), "hierarchy or Git"},
		{"incomplete hierarchy", Verify, func(v *Result) { v.Handoffs = v.Handoffs[:1] }, "hierarchy or Git"},
		{"surplus handoff", Verify, func(v *Result) { v.Handoffs = append(v.Handoffs, Handoff{Manager: "b"}) }, "hierarchy or Git"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := closedResult(tc.mandate)
			tc.change(&v)
			err := validateClosed(v, closedRequest(tc.mandate), "binding", closedManagers)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want a refusal containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestValidateClosedRefusesWithoutCheckedPreparation(t *testing.T) {
	if err := ValidateClosed(closedResult(Verify), closedRequest(Verify)); err == nil || !strings.Contains(err.Error(), "checked Project preparation") {
		t.Fatal("a result validated without checked preparation:", err)
	}
}
