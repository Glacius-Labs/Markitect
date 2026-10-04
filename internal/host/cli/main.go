package cli

import (
	"fmt"
	"io"
	"runtime"

	"github.com/Glacius-Labs/Markitect/internal/host"
)

// Run dispatches a Markitect CLI invocation and returns its process exit code.
func Run(args []string, out, errout io.Writer) int {
	if len(args) == 0 {
		printUsage(errout)
		return 2
	}
	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		if len(args) == 1 {
			printUsage(out)
			return 0
		}
		if args[0] == "help" && len(args) == 2 {
			return Run([]string{args[1], "--help"}, out, errout)
		}
		fmt.Fprintln(errout, "help accepts at most one command")
		return 2
	}
	command := args[0]
	allowed, known := commandFlags(command)
	if !known {
		fmt.Fprintf(errout, "unknown command %q\n", command)
		return 2
	}
	if command == "version" {
		if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
			fmt.Fprintln(out, "usage: markitect version")
			return 0
		}
		if len(args) != 1 {
			fmt.Fprintln(errout, "version accepts no arguments")
			return 2
		}
		fmt.Fprintf(out, "Markitect %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
		return 0
	}
	o, code, done := parseOptions(command, args, allowed, out, errout)
	if done {
		return code
	}
	emit := func(value any) int {
		data, err := host.YAML(value)
		if err != nil {
			fmt.Fprintln(errout, err)
			return 2
		}
		if _, err = out.Write(data); err != nil {
			fmt.Fprintln(errout, err)
			return 2
		}
		return 0
	}
	fail := func(err error) int { fmt.Fprintln(errout, err); return 2 }
	return dispatchCommand(command, o, out, errout, emit, fail)
}
