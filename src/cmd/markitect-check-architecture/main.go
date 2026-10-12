package main

import (
	"os"

	"github.com/Glacius-Labs/Markitect/src/internal/application/architecturecli"
)

func main() { os.Exit(architecturecli.Run(os.Args[1:], os.Stdout, os.Stderr)) }
