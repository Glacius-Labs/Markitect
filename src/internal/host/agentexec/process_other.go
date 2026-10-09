//go:build !linux && !windows

package agentexec

import (
	"errors"
	"os/exec"
)

type unsupportedProcessTree struct{}

func newProcessTreeGuard() (processTreeGuard, error) {
	return nil, errors.New("runner process-tree control is unsupported on this platform")
}

func (*unsupportedProcessTree) prepare(*exec.Cmd) error { return nil }
func (*unsupportedProcessTree) attach(*exec.Cmd) error  { return nil }
func (*unsupportedProcessTree) terminate() error        { return nil }
func (*unsupportedProcessTree) close() error            { return nil }
