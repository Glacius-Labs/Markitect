// Command markitect-agentd is the session helper inside every Markitect
// Agent container.
//
//	markitect-agentd serve [--socket PATH]   serve the Host on standard input and output
//	markitect-agentd mcp [--socket PATH]     bridge a CLI's stdio MCP client to serve
//	markitect-agentd version                 print the build version
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/Glacius-Labs/Markitect/internal/agentd"
)

// version is injected at build time with -ldflags "-X main.version=...".
var version = "development"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	os.Exit(run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	verb := "serve"
	if len(args) > 0 {
		verb, args = args[0], args[1:]
	}
	flags := flag.NewFlagSet("markitect-agentd "+verb, flag.ContinueOnError)
	flags.SetOutput(stderr)
	socket := flags.String("socket", agentd.DefaultSocketPath, "Unix socket for MCP bridges")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return 2
	}
	var err error
	switch verb {
	case "serve":
		err = agentd.Serve(ctx, stdin, stdout, agentd.Config{Version: version, SocketPath: *socket})
	case "mcp":
		err = agentd.Bridge(ctx, *socket, stdin, stdout)
	case "version":
		_, err = fmt.Fprintln(stdout, version)
	default:
		fmt.Fprintf(stderr, "markitect-agentd: unknown command %q\n", verb)
		return 2
	}
	if err != nil {
		fmt.Fprintf(stderr, "markitect-agentd %s: %v\n", verb, err)
		return 1
	}
	return 0
}
