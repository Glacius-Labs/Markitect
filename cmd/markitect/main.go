// Command markitect is the Markitect tool: show, table, topic, questions and
// diff to look; check to check; and version and licenses about the tool
// itself. It delegates everything to internal/cli.
package main

import (
	"os"

	"github.com/Glacius-Labs/Markitect/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
