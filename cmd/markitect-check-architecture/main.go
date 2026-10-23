package main

import (
	"os"

	"github.com/Glacius-Labs/Markitect/internal/tooling/architecturecli"
)

func main() { os.Exit(architecturecli.Run(os.Args[1:], os.Stdout, os.Stderr)) }
