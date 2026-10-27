// Command markitect is the Markitect command line: check, context, docs,
// apply, version and licenses. It delegates everything to internal/bootstrap.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Glacius-Labs/Markitect/internal/bootstrap"
)

func main() {
	// The first interrupt or termination signal cancels the context, so an Apply
	// stops every turn and container and still writes its evidence and report.
	// A second signal then takes its default effect.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ctx.Done()
		stop()
	}()
	defer stop()
	os.Exit(bootstrap.Run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}
