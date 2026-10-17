package main

import (
	"os"

	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"github.com/Glacius-Labs/Markitect/src/internal/application/projectcli"
)

// The product binary dispatches the verb table using independent build metadata.
func main() {
	os.Exit(projectcli.Main(os.Args[1:], os.Stdout, os.Stderr, buildinfo.Version()))
}
