// Command markitect is the Markitect tool: init, check, show, table, new,
// move, apply, topic, questions, status, mark, compare, figures, reports,
// settings, version and licenses. It delegates everything to internal/cli.
package main

import (
	"os"

	"github.com/Glacius-Labs/Markitect/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
