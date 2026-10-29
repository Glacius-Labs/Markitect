package run

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// TestResultCommitsAreNoAgentCommits checks that the trailers of the host's
// Result commits are plain to the agents' trailer parser: every Run starts
// from a landing commit, and a submission or review refuses a branch tip
// with an unknown Markitect trailer.
func TestResultCommitsAreNoAgentCommits(t *testing.T) {
	r := &Result{Run: "r1", State: StateKept, Target: "main", Start: "abc", Full: true, impact: []string{"def:catalog/book", "file:src/book.go"}}
	for _, message := range []string{resultMessage(r, ""), resultMessage(r, "def")} {
		tr, err := runremote.ParseMessage(message)
		if err != nil || tr.Kind != runremote.Plain {
			t.Fatalf("%q reads as %+v, %v", message, tr, err)
		}
		if got := trailers(message)[trailerRun]; len(got) != 1 || got[0] != "r1" {
			t.Fatalf("%q: %v", message, trailers(message))
		}
	}
	if got := trailers(resultMessage(r, ""))[trailerImpact]; len(got) != 2 {
		t.Fatalf("impact %v", got)
	}
}
