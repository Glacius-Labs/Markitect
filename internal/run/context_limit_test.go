package run

import (
	"fmt"
	"strings"
	"testing"
)

// A turn whose input the provider channel cannot carry is not repeated, which
// would only grow its input: the run ends Incomplete at once, naming why.
func TestAContextLimitEndsTheRunWithoutRetry(t *testing.T) {
	f := newFixture(t)
	turns := 0
	f.agents.script = func(c *call) error {
		turns++
		return fmt.Errorf("%w: the prompt of 2000000 bytes exceeds the 1048576 bytes the provider channel carries", ErrContextLimit)
	}
	r := f.open()
	r.Pass(implementSpec())
	out := f.finish(r, "")
	if out.Status != Incomplete || !strings.Contains(out.Reason, "exceeds the 1048576 bytes") || turns != 1 {
		t.Fatalf("status %s reason %q after %d turns", out.Status, out.Reason, turns)
	}
	if len(out.Problems) != 1 || !strings.Contains(out.Problems[0].Text, "2000000 bytes") {
		t.Fatalf("problems %+v", out.Problems)
	}
}
