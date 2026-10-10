package main

import (
	"os"

	hostcli "github.com/Glacius-Labs/Markitect/src/internal/host/cli"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectcli"
)

// The product binary dispatches the verb table. It links the legacy cli
// package only for its release-injected version (-X .../host/cli.version).
func main() {
	os.Exit(projectcli.Main(os.Args[1:], os.Stdout, os.Stderr, hostcli.Version()))
}
