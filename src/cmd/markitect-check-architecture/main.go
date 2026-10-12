package main

import (
	"os"

	"github.com/Glacius-Labs/Markitect/src/internal/host/architecturecli"
)

func main() { os.Exit(architecturecli.Run(os.Args[1:], os.Stdout, os.Stderr)) }
