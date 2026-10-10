//go:build !windows

package projectrun

import (
	"os/exec"
	"syscall"
)

// runCheckProcess runs a check as the leader of its own process group. Its
// timeout and its completion stop the whole group, so a descendant can neither
// outlive the check nor hold its output pipes open past the check timeout.
func runCheckProcess(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		return cmd.Process.Kill()
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	err := cmd.Wait()
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	return err
}
