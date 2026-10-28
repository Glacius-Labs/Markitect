// Command markitect-remote is the program of a Run's remote container
// (internal/runremote/README.md): it seeds the bare repository, serves it to
// the Managers, decides every push as its pre-receive hook and exports the
// result.
//
//	markitect-remote seed|serve|hook|ready|export|log [--root DIR]
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Glacius-Labs/Markitect/internal/adapters/remoteserver"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(remoteserver.Main(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
