package run

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

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
