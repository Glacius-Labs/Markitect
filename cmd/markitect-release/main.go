package main

import (
	"os"

	"github.com/Glacius-Labs/Markitect/internal/host/releasecli"
)

func main() { os.Exit(releasecli.Run(os.Args[1:], os.Stdout, os.Stderr)) }
