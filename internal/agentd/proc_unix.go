//go:build unix

package agentd

import (
	"errors"
	"os/exec"
	"syscall"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

// ownProcessGroup starts the provider as the leader of a new process group,
// so stopping it reaches every process it started.
func ownProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// terminate asks the provider's whole process group to end.
func terminate(cmd *exec.Cmd) { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM) }

// kill ends the provider's whole process group.
func kill(cmd *exec.Cmd) { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }

// reapProcessGroup kills processes the provider left behind after its exit.
func reapProcessGroup(pid int) { _ = syscall.Kill(-pid, syscall.SIGKILL) }

func exitOf(cmd *exec.Cmd, waitErr error) frame.Exited {
	state := cmd.ProcessState
	if state == nil {
		return frame.Exited{Code: -1, Signal: "unknown: " + waitErr.Error()}
	}
	if status, ok := state.Sys().(syscall.WaitStatus); ok && status.Signaled() {
		return frame.Exited{Code: -1, Signal: status.Signal().String()}
	}
	var exitErr *exec.ExitError
	if waitErr != nil && !errors.As(waitErr, &exitErr) && !errors.Is(waitErr, exec.ErrWaitDelay) {
		return frame.Exited{Code: state.ExitCode(), Signal: "wait: " + waitErr.Error()}
	}
	return frame.Exited{Code: state.ExitCode()}
}
