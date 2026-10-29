package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

type landedRuns struct{ r *run.Result }

func (l landedRuns) Run(context.Context, RunOptions) (*run.Result, error) { return l.r, nil }

// The landed page cannot say what the cleanup after landing left, so the
// command says it on stderr, with the page or the JSON on stdout.
func TestTheCommandReportsTheCleanupAfterLanding(t *testing.T) {
	r := &run.Result{Run: "r1", Command: "apply", State: run.StateLanded, Outcome: run.Landed,
		AfterLanding: []string{"the landed Run's branch markitect/runs/r1 could not be deleted: refused"}}
	for _, args := range [][]string{{"apply"}, {"apply", "--json"}} {
		var out, errOut bytes.Buffer
		c := CLI{Runs: landedRuns{r}, Stdout: &out, Stderr: &errOut}
		if code := c.Run(context.Background(), args); code != OK {
			t.Fatalf("%v: exit %d: %s", args, code, errOut.String())
		}
		if !strings.Contains(errOut.String(), "after landing, the landed Run's branch markitect/runs/r1 could not be deleted") {
			t.Fatalf("%v: stderr %q", args, errOut.String())
		}
	}
}
