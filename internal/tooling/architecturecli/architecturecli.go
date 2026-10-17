// Package architecturecli exposes the repository-owned architecture gate to
// its thin command-line entrypoint.
package architecturecli

import (
	"flag"
	"fmt"
	"io"

	"github.com/Glacius-Labs/Markitect/internal/tooling/architecture"
)

// Run checks repository import policy and writes deterministic diagnostics.
func Run(args []string, out, errout io.Writer) int {
	flags := flag.NewFlagSet("markitect-check-architecture", flag.ContinueOnError)
	flags.SetOutput(errout)
	repo := flags.String("repo", ".", "explicit repository root")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(errout, "unexpected positional arguments")
		return 2
	}
	violations, err := architecture.CheckRepository(*repo)
	if err != nil {
		fmt.Fprintln(errout, err)
		return 2
	}
	for _, violation := range violations {
		fmt.Fprintln(out, violation.String())
	}
	if len(violations) != 0 {
		return 1
	}
	fmt.Fprintln(out, "architecture imports: passed")
	return 0
}
