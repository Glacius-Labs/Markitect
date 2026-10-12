package main

import (
	"os"

	"github.com/Glacius-Labs/Markitect/src/internal/runtime/exchangecli"
)

func main() { os.Exit(exchangecli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }
