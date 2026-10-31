package run

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// A checker gets the required checks' results on the submission it reviews,
// from the push that brought it (DEC-087).
func TestTheCheckEvidenceOfASubmission(t *testing.T) {
	s := &schedule{plan: &runremote.Plan{Checks: []project.Check{{Name: "build"}, {Name: "test"}}}}
	push := func(commit string, checks ...runremote.CheckResult) Push {
		return Push{Updates: []runremote.Update{{New: commit}}, Decision: runremote.Decision{Accepted: true, Kind: "submission", Checks: checks}}
	}
	passing := []Push{push("old", runremote.CheckResult{Name: "build", Exit: 1}),
		push("new", runremote.CheckResult{Name: "build", Passed: true}, runremote.CheckResult{Name: "test", Passed: true})}
	if got := s.checkEvidence(passing, nil, "new"); !strings.Contains(got, `"build" passed.`) || !strings.Contains(got, `"test" passed.`) || strings.Contains(got, "exited") {
		t.Fatalf("the newer submission's evidence:\n%s", got)
	}

	long := "é" + strings.Repeat("x", evidenceTail) + "the end"
	failing := []Push{push("sub", runremote.CheckResult{Name: "build", Exit: 2, Output: long})}
	got := s.checkEvidence(failing, nil, "sub")
	if !strings.Contains(got, `"build" exited with 2.`) || !strings.Contains(got, "the end") || strings.Contains(got, "é") ||
		!strings.Contains(got, "written by the code under review") || !strings.Contains(got, "the merged state must pass them") || !utf8.ValidString(got) {
		t.Fatalf("the failing evidence:\n%s", got)
	}

	if got := s.checkEvidence(nil, errors.New("gone"), "sub"); !strings.Contains(got, "could not read") {
		t.Fatalf("without the log: %s", got)
	}
	if got := s.checkEvidence(passing, nil, "other"); !strings.Contains(got, "holds no required checks' results") {
		t.Fatalf("without the push: %s", got)
	}
	if got := (&schedule{plan: &runremote.Plan{}}).checkEvidence(passing, nil, "new"); got != "" {
		t.Fatalf("without checks: %s", got)
	}
}

// The tasks say what a verdict covers (DEC-088): each item's whole
// Definition with its examples and links, the Manager's own files, and a
// parent that integrates its children's work by using it.
func TestTheTasksSayWhatAVerdictCovers(t *testing.T) {
	s := &schedule{plan: &runremote.Plan{Base: "base0000000000"}}
	root := &node{name: ".", branch: "root", from: "base0000000000", submission: "sub", items: make([]agentport.Item, 1)}
	child := &node{name: "materials", branch: "managers/materials", from: "base0000000000", parent: root, submission: "sub2", items: make([]agentport.Item, 2),
		released: true, approved: true, approval: "appr"}
	root.children = []*node{child}

	maker := s.makerTask(root).task
	for _, want := range []string{"A verdict covers the item's whole Definition, its examples included, and the Definitions it links to", "names how each example is met",
		"Merge it, review it and integrate it before you submit: use it, and do not build a child's behavior a second time in your own files."} {
		if !strings.Contains(maker, want) {
			t.Errorf("maker task lacks %q:\n%s", want, maker)
		}
	}
	review := s.reviewTask(root).task
	for _, want := range []string{"It integrates the work of materials: dispute it when one of the Manager's own files builds a child's behavior a second time",
		"Judge the Manager's own files against its items, and each item's whole Definition with its examples and the Definitions it links to."} {
		if !strings.Contains(review, want) {
			t.Errorf("Root's review task lacks %q:\n%s", want, review)
		}
	}
	if strings.Contains(review, "parent's integration") {
		t.Errorf("Root has no parent:\n%s", review)
	}
	if review := s.reviewTask(child).task; !strings.Contains(review, "whether the rest of the program uses those files yet is its parent's integration.") || strings.Contains(review, "It integrates") {
		t.Errorf("the child's review task:\n%s", review)
	}
	if v := s.verifyTask(child).task; !strings.Contains(v, "the evidence for conforms names how each example is met") {
		t.Errorf("verify task:\n%s", v)
	}
}

// A turn that continues an agent gets only what brought it about: a maker
// the children's approved work and its messages, a checker the new
// submission with the checks' results on it; the task its first turn
// received stays out.
func TestALaterTurnsEventLeavesOutTheFirstTurnsTask(t *testing.T) {
	s := &schedule{plan: &runremote.Plan{Base: "base0000000000"}}
	root := &node{name: ".", branch: "root", from: "base0000000000", submission: "sub", items: make([]agentport.Item, 1),
		evidence: "The required checks on the branch's full state at this submission:\n- \"build\" passed.",
		inbox:    []agentport.Message{{ID: "m1", Kind: agentport.Review, To: ".", Text: "Your checker rejected submission sub."}}}
	child := &node{name: "materials", branch: "managers/materials", from: "base0000000000", parent: root, released: true, approved: true, approval: "appr"}
	root.children = []*node{child}

	maker := s.makerTask(root)
	if maker.event != "Your working children's approved work: materials at appr (branch managers/materials). Merge it, review it and integrate it before you submit: use it, and do not build a child's behavior a second time in your own files. The messages delivered with this turn need your action first." {
		t.Errorf("maker event:\n%s", maker.event)
	}
	if !strings.HasPrefix(maker.task, "Realize the change for Manager .") || !strings.HasSuffix(maker.task, maker.event) {
		t.Errorf("maker task:\n%s", maker.task)
	}
	review := s.reviewTask(root)
	if !strings.HasPrefix(review.event, "Review submission sub of Manager . on branch root") || !strings.Contains(review.event, "It integrates the work of materials") ||
		!strings.HasSuffix(review.event, `"build" passed.`) || strings.Contains(review.event, "Judge the Manager's own files") {
		t.Errorf("review event:\n%s", review.event)
	}
	if !strings.Contains(review.task, "Judge the Manager's own files") || !strings.HasSuffix(review.task, `"build" passed.`) {
		t.Errorf("review task:\n%s", review.task)
	}
	if v := s.verifyTask(child); v.event != "Your last turn pushed no verdicts on branch managers/materials. Push them as one commit without file changes." {
		t.Errorf("verify event: %s", v.event)
	}
}

// A verify checker gets the required checks the host ran on the start: one
// that did not start says why in the remote's words, and checks that could
// not run say so.
func TestTheStartEvidenceOfVerify(t *testing.T) {
	s := &schedule{plan: &runremote.Plan{Base: "base0000000000", Checks: []project.Check{{Name: "build"}}}}
	if got := s.startEvidence(); got != "" {
		t.Fatalf("before the run: %q", got)
	}
	s.startChecks = &startChecks{results: []runremote.CheckResult{{Name: "build", Exit: -1, Output: "[the check's directory realization does not exist in this state]\n"}}}
	got := s.startEvidence()
	if !strings.Contains(got, "run by the host before you started (commit base00000000)") ||
		!strings.Contains(got, `"build" did not start: [the check's directory realization does not exist in this state]`) ||
		strings.Contains(got, "written by the code under review") || strings.Contains(got, "exited with -1") {
		t.Fatalf("a check that did not start:\n%s", got)
	}
	s.startChecks = &startChecks{err: errors.New("the remote is gone")}
	if got := s.startEvidence(); !strings.Contains(got, "could not run on this state: the remote is gone") {
		t.Fatalf("checks that could not run: %s", got)
	}
}
