package agentexec

import (
	"os/exec"
)

type processTreeGuard interface {
	prepare(*exec.Cmd) error
	attach(*exec.Cmd) error
	terminate() error
	close() error
}
