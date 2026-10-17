//go:build !linux

package agentd

import (
	"errors"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

func listProcesses(int) ([]frame.Process, error) {
	return nil, errors.New("no process table on this platform; agentd runs in Linux containers")
}
