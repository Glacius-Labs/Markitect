//go:build !unix

package agentd

import (
	"os/exec"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

// agentd runs only inside Linux containers. These fallbacks keep the package
// buildable and testable on other development hosts; they stop only the
// provider process itself, not its descendants.

func ownProcessGroup(*exec.Cmd) {}

func terminate(cmd *exec.Cmd) { _ = cmd.Process.Kill() }

func kill(cmd *exec.Cmd) { _ = cmd.Process.Kill() }

func reapProcessGroup(int) {}

func exitOf(cmd *exec.Cmd, waitErr error) frame.Exited {
	if cmd.ProcessState == nil {
		return frame.Exited{Code: -1, Signal: "unknown: " + waitErr.Error()}
	}
	return frame.Exited{Code: cmd.ProcessState.ExitCode()}
}
