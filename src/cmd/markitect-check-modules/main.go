package main

import (
	"os"

	"github.com/Glacius-Labs/Markitect/src/internal/host/modulecli"
)

func main() { os.Exit(modulecli.Run(os.Args[1:], os.Stdout, os.Stderr)) }
