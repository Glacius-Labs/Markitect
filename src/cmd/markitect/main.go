package main

import (
	"os"

	hostcli "github.com/Glacius-Labs/Markitect/src/internal/host/cli"
)

func main() { os.Exit(hostcli.Run(os.Args[1:], os.Stdout, os.Stderr)) }
