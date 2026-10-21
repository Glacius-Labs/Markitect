package agentd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

func hold(ctx context.Context, cfg HoldConfig) error {
	// As the container's init it receives only signals it handles; ignoring
	// them leaves the engine's kill as the only way to end it early.
	signal.Ignore(syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGPIPE)
	ticker := time.NewTicker(cfg.Poll)
	defer ticker.Stop()
	busy := time.Now()
	for {
		reapChildren()
		if processes, err := listProcesses(os.Getpid()); err != nil || len(live(processes)) > 0 {
			busy = time.Now()
		}
		if time.Since(busy) >= cfg.Idle {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// reapChildren collects every exited child, including the orphans the kernel
// gave to the init process.
func reapChildren() {
	for {
		var status syscall.WaitStatus
		pid, err := syscall.Wait4(-1, &status, syscall.WNOHANG, nil)
		if pid <= 0 || err != nil {
			return
		}
	}
}

func reap(timeout time.Duration) ([]frame.Process, error) {
	deadline := time.Now().Add(timeout)
	for {
		// Every process the caller may signal, which excludes init and the
		// caller itself.
		_ = syscall.Kill(-1, syscall.SIGKILL)
		processes, err := listProcesses(os.Getpid())
		if err != nil {
			return nil, err
		}
		remaining := live(processes)
		if len(remaining) == 0 || time.Now().After(deadline) {
			return remaining, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
}
