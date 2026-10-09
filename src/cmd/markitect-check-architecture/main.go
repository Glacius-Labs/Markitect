package main

import (
	"github.com/Glacius-Labs/Markitect/src/internal/host"
	"os"
)

func main() { os.Exit(host.RunArchitectureCheck(os.Args[1:], os.Stdout, os.Stderr)) }
