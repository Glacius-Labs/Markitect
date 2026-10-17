package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/Glacius-Labs/Markitect/internal/bootstrap"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	os.Exit(bootstrap.Run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}
