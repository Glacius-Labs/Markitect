package agentd_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

// census asks agentd for the process table and reports whether pid is a
// live process in it. A zombie has ended; it only awaits its parent's wait.
func (h *harness) census(pid int) bool {
	h.t.Helper()
	h.sendControl(frame.ControlMessage{Type: frame.TypeCensus, Census: &frame.Census{}})
	for _, p := range h.control(frame.TypeCensus).Census.Processes {
		if p.PID == pid {
			return p.State != "Z"
		}
	}
	return false
}

func TestCensusShowsABackgroundToolProcessUntilItEnds(t *testing.T) {
	h := newHarness(t)
	h.sendControl(frame.ControlMessage{Type: frame.TypeStart, Start: &frame.Start{
		Argv: []string{os.Args[0]}, Dir: t.TempDir(), Env: []string{"AGENTD_TEST_PROVIDER=background"},
	}})
	provider := h.control(frame.TypeStarted).Started.PID
	line := h.output(frame.ProviderOut, "\n")
	var child int
	if _, err := fmt.Sscanf(line, "child %d", &child); err != nil {
		t.Fatalf("child pid from %q: %v", line, err)
	}
	if !h.census(provider) || !h.census(child) {
		t.Fatal("the census misses the provider or its running child")
	}
	deadline := time.Now().Add(10 * time.Second)
	for h.census(child) {
		if time.Now().After(deadline) {
			t.Fatalf("child %d still listed after it ended", child)
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !h.census(provider) {
		t.Fatal("the idle provider disappeared from the census")
	}
	h.send(frame.Frame{Channel: frame.ProviderIn})
	h.control(frame.TypeExited)
}
