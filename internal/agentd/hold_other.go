//go:build !linux

package agentd

import (
	"context"
	"errors"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

// errLinuxOnly refuses holding and reaping where no Linux process table and
// signals exist; Agent containers are Linux.
var errLinuxOnly = errors.New("holding and reaping a container needs Linux")

func hold(context.Context, HoldConfig) error { return errLinuxOnly }

func reap(time.Duration) ([]frame.Process, error) { return nil, errLinuxOnly }
