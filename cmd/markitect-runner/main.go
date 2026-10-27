// Command markitect-runner is the entrypoint of the Linux runner image
// (docs/design/linux-runner.md). It clones the read-only checkout, runs
// markitect apply against the clone and writes the result for hand-back.
//
//	markitect-runner --base COMMIT [--model PATH] [--dependency PATH]...
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Glacius-Labs/Markitect/internal/adapters/runner"
)

func main() {
	// A stop request cancels the context; the runner passes it on to Apply as
	// SIGTERM and still writes its hand-back record.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(runner.Main(ctx, os.Args[1:], os.Stderr))
}
