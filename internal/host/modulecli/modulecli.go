// Package modulecli is the process-facing interface for explicit Host Module checks.
package modulecli

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/Glacius-Labs/Markitect/internal/host"
)

// Run dispatches one read-only Host Module check invocation.
func Run(args []string, out, errout io.Writer) int {
	flags := flag.NewFlagSet("markitect-check-modules", flag.ContinueOnError)
	flags.SetOutput(errout)
	repo := flags.String("repo", ".", "repository root")
	revision := flags.String("revision", "", "optional immutable Git revision")
	hooks := flags.String("hooks", "", "exact repository-relative git hooks config path")
	pipelines := flags.String("pipelines", "", "exact repository-relative pipelines config path")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(errout, "unexpected positional arguments")
		return 2
	}
	if *hooks == "" && *pipelines == "" {
		fmt.Fprintln(errout, "at least one of --hooks or --pipelines is required")
		return 2
	}
	report, err := host.CheckModules(host.ModuleChecksOptions{
		Root: *repo, Revision: *revision, HooksConfigPath: *hooks, PipelinesPath: *pipelines,
	})
	if err != nil {
		fmt.Fprintln(errout, err)
		return 2
	}
	data, err := host.YAML(report)
	if err != nil {
		fmt.Fprintln(errout, err)
		return 2
	}
	n, err := out.Write(data)
	if err != nil {
		fmt.Fprintln(errout, err)
		return 2
	}
	if n != len(data) {
		fmt.Fprintln(errout, "short write while printing module check report")
		return 2
	}
	if report.Status != host.ModuleChecksPassed {
		return 1
	}
	return 0
}
