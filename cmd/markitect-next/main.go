// Command markitect-next is the rebuilt Markitect tool while it grows beside
// markitect: check. It delegates everything to internal/cli.
package main

import (
	"os"

	"github.com/Glacius-Labs/Markitect/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
