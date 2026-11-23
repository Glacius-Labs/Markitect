// Command markitect is the Markitect tool: show, table, topic, questions and
// compare to look; check and status to check; new, apply and move to change
// the spec; confirm; and init, version and licenses to set up. It delegates
// everything to internal/cli.
package main

import (
	"os"

	"github.com/Glacius-Labs/Markitect/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
