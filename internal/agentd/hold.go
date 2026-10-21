package agentd

import (
	"context"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

// HoldConfig bounds the init process of a reusable Agent container. Idle is
// how long the container may stay without any other process before the
// holder ends it; Poll is how often it reaps orphans and looks.
type HoldConfig struct {
	Idle time.Duration
	Poll time.Duration
}

// Hold runs as the init process of a reusable Agent container. It keeps the
// container alive between the agentd sessions that run in it through the
// engine's exec, reaps every orphaned process, ignores the signals other
// processes of the container may send it and returns once the container held
// no other process for cfg.Idle, which ends the container. It runs under its
// own user, so the Agents' processes can neither signal nor trace it.
func Hold(ctx context.Context, cfg HoldConfig) error {
	if cfg.Idle <= 0 {
		cfg.Idle = 30 * time.Minute
	}
	if cfg.Poll <= 0 {
		cfg.Poll = time.Second
	}
	return hold(ctx, cfg)
}

// ReapTimeout bounds Reap by default.
const ReapTimeout = 5 * time.Second

// Reap ends every process of the container that the caller may signal, except
// the container's init and the caller itself, and returns the processes that
// still live once none is left or timeout passed. Zombies, which can write
// nothing and await their reaper, do not count. An empty result is the
// observed cessation of every Agent process of the container; an error means
// it could not be observed.
func Reap(timeout time.Duration) ([]frame.Process, error) {
	if timeout <= 0 {
		timeout = ReapTimeout
	}
	return reap(timeout)
}

// live are the processes that can still run: not the init process and not
// a zombie or dead entry, unless other threads of a zombie leader's group
// still run.
func live(processes []frame.Process) []frame.Process {
	var out []frame.Process
	for _, p := range processes {
		if p.PID != 1 && (p.State != "Z" && p.State != "X" || p.Threads > 1) {
			out = append(out, p)
		}
	}
	return out
}
