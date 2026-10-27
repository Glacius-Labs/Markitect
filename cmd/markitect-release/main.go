// Command markitect-release prepares local Markitect distributions: source
// packaging, bundling and bundle installation, plus the version and license
// notices. It never calls GitHub or publishes a release.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
)

func main() { os.Exit(runMain(os.Args[1:], os.Stdout, os.Stderr)) }

// runMain executes the release command and returns its process exit code.
func runMain(args []string, out, errout io.Writer) int {
	if err := run(args, out, errout); err != nil {
		fmt.Fprintln(errout, err)
		return 2
	}
	return 0
}

func run(args []string, out, errout io.Writer) error {
	if len(args) == 0 {
		usage(errout)
		return errors.New("missing command")
	}
	switch args[0] {
	case "package", "bundle", "install", "version", "licenses":
		return runLocal(args[0], args[1:], out, errout)
	case "-h", "-help", "--help":
		usage(errout)
		return nil
	}
	usage(errout)
	return fmt.Errorf("unknown command %q", args[0])
}

func usage(errout io.Writer) {
	fmt.Fprintln(errout, "Usage of markitect-release:")
	fmt.Fprintln(errout, "  package, bundle, install, version, licenses: local release preparation")
}
