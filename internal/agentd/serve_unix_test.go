//go:build unix

package agentd_test

import (
	"errors"
	"fmt"
	"syscall"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

// Process groups exist only on Unix hosts; agentd runs in Linux containers.
func TestStopEndsTheWholeProcessTree(t *testing.T) {
	h := newHarness(t)
	h.start("stubborn-tree")
	line := h.output(frame.ProviderOut, "\n")
	var child int
	if _, err := fmt.Sscanf(line, "child %d", &child); err != nil {
		t.Fatalf("child pid from %q: %v", line, err)
	}
	h.sendControl(frame.ControlMessage{Type: frame.TypeStop, Stop: &frame.Stop{GraceMillis: 200}})
	if exit := h.control(frame.TypeExited); exit.Exited.Signal != "killed" {
		t.Fatalf("exit = %+v, want a kill after the ignored polite signal", exit.Exited)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if err := syscall.Kill(child, 0); errors.Is(err, syscall.ESRCH) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("grandchild %d survived the stop", child)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
