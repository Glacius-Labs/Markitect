//go:build linux

package agentexec

import (
	"errors"
	"os/exec"
	"syscall"
)

type linuxProcessTree struct {
	groupID int
}

func newProcessTreeGuard() (processTreeGuard, error) {
	return &linuxProcessTree{}, nil
}

func (g *linuxProcessTree) prepare(cmd *exec.Cmd) error {
	if cmd.SysProcAttr != nil {
		return errors.New("runner process attributes are already configured")
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return nil
}

func (g *linuxProcessTree) attach(cmd *exec.Cmd) error {
	if cmd.Process == nil || cmd.Process.Pid <= 0 {
		return errors.New("runner process identity is unavailable")
	}
	g.groupID = cmd.Process.Pid
	return nil
}

func (g *linuxProcessTree) terminate() error {
	if g.groupID == 0 {
		return nil
	}
	err := syscall.Kill(-g.groupID, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	if err != nil {
		return errors.New("runner process group could not be stopped")
	}
	return nil
}

func (g *linuxProcessTree) close() error {
	return g.terminate()
}
