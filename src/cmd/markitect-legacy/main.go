// Command markitect-legacy hosts the v0.13 Project/Domain command tree until
// ARCH-09 removes it. The product binary dispatches the verb table instead.
package main

import (
	"os"

	hostcli "github.com/Glacius-Labs/Markitect/src/internal/host/cli"
)

func main() { os.Exit(hostcli.Run(os.Args[1:], os.Stdout, os.Stderr)) }
