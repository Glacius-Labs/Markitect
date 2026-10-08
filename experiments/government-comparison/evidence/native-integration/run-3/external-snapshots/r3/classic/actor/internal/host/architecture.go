package host

import (
	"flag"
	"fmt"
	"github.com/Glacius-Labs/Markitect/internal/tooling/architecture"
	"io"
)

// RunArchitectureCheck is the read-only runtime for the repository-owned
// import policy. It interprets implementation imports outside the semantic kernel.
func RunArchitectureCheck(args []string, out, errout io.Writer) int {
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
	edges, err := architecture.Inspect(*repo)
	if err != nil {
		fmt.Fprintln(errout, err)
		return 2
	}
	violations := architecture.Check(edges)
	for _, v := range violations {
		fmt.Fprintln(out, v.String())
	}
	if len(violations) != 0 {
		return 1
	}
	fmt.Fprintln(out, "architecture imports: passed")
	return 0
}
