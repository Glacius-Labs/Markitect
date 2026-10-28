// Command markitect is the Markitect command line: init, check, lock, refs,
// graph, impact, lint, version and licenses. It delegates everything to
// internal/bootstrap.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Glacius-Labs/Markitect/internal/bootstrap"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(bootstrap.Run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}
