package main

import (
	"os"

	"github.com/Glacius-Labs/Markitect/internal/tooling/releasecli"
)

func main() { os.Exit(releasecli.Run(os.Args[1:], os.Stdout, os.Stderr)) }
