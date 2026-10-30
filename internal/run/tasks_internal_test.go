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

	maker := s.makerTask(root)
	for _, want := range []string{"A verdict covers the item's whole Definition, its examples included, and the Definitions it links to", "names how each example is met",
		"Merge it, review it and integrate it before you submit: use it, and do not build a child's behavior a second time in your own files."} {
		if !strings.Contains(maker, want) {
			t.Errorf("maker task lacks %q:\n%s", want, maker)
		}
	}
	review := s.reviewTask(root)
	for _, want := range []string{"It integrates the work of materials: dispute it when one of the Manager's own files builds a child's behavior a second time",
		"Judge the Manager's own files against its items, and each item's whole Definition with its examples and the Definitions it links to."} {
		if !strings.Contains(review, want) {
			t.Errorf("Root's review task lacks %q:\n%s", want, review)
		}
	}
	if strings.Contains(review, "parent's integration") {
		t.Errorf("Root has no parent:\n%s", review)
	}
	if review := s.reviewTask(child); !strings.Contains(review, "whether the rest of the program uses those files yet is its parent's integration.") || strings.Contains(review, "It integrates") {
		t.Errorf("the child's review task:\n%s", review)
	}
	if v := s.verifyTask(child); !strings.Contains(v, "the evidence for conforms names how each example is met") {
		t.Errorf("verify task:\n%s", v)
	}
}
