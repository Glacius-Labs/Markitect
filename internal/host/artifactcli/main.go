// markitect-check-artifacts validates explicit ownership for configured
// repository roots against Markitect's normalized project and rendered outputs.
package artifactcli

import (
	"flag"
	"fmt"
	"io"

	"github.com/Glacius-Labs/Markitect/internal/artifactcoverage"
	"github.com/Glacius-Labs/Markitect/internal/host"
)

// Run checks configured artifact ownership and returns its process exit code.
func Run(args []string, out, errout io.Writer) int {
	flags := flag.NewFlagSet("markitect-check-artifacts", flag.ContinueOnError)
	flags.SetOutput(errout)
	repo := flags.String("repo", ".", "repository root")
	config := flags.String("config", "markitect-artifacts.yaml", "repository-relative artifact coverage config")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(errout, "unexpected positional arguments")
		return 2
	}
	report, err := artifactcoverage.Check(*repo, *config)
	if err != nil {
		fmt.Fprintln(errout, err)
		return 2
	}
	data, err := host.YAML(report)
	if err != nil {
		fmt.Fprintln(errout, err)
		return 2
	}
	if _, err := out.Write(data); err != nil {
		fmt.Fprintln(errout, err)
		return 2
	}
	if report.Status != "passed" {
		return 1
	}
	return 0
}
