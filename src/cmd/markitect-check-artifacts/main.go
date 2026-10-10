package main

import (
	"os"

	"github.com/Glacius-Labs/Markitect/src/internal/host/artifactcli"
)

func main() { os.Exit(artifactcli.Run(os.Args[1:], os.Stdout, os.Stderr)) }
