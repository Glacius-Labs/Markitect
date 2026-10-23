package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"io"
	"path/filepath"
)

type ApplyRequest struct {
	Selection        project.InputSelection
	RunConfig, Build string
}
type ApplyReport struct {
	FormatVersion int          `json:"formatVersion"`
	Operation     string       `json:"operation"`
	Status        model.Status `json:"status"`
	Error         string       `json:"error,omitempty"`
}
type ApplyFunc func(context.Context, ApplyRequest) (ApplyReport, error)

func (c *Command) applyVerb(ctx context.Context, args []string, stdout, stderr io.Writer, emit func(any) int) int {
	const verb = "apply"
	incomplete := func(reason string) int {
		if emit(map[string]string{"operation": verb, "status": "not_completed", "reason": reason}) != 0 {
			return 3
		}
		return 3
	}
	if c.apply == nil {
		return incomplete("Execution is unavailable because its concrete execution/publication ports are not configured.")
	}
	flags := flag.NewFlagSet(verb, flag.ContinueOnError)
	flags.SetOutput(stderr)
	config := flags.String("run-config", "", "explicit absolute invocation configuration")
	var repo, manifest, runtimeProfile string
	var dependencies manifestList
	flags.StringVar(&repo, "repo", ".", "repository root")
	flags.StringVar(&manifest, "model", ".markitect/model.yaml", "explicit model.yaml")
	flags.StringVar(&runtimeProfile, "runtime-profile", "", "explicit absolute Runtime profile document")
	flags.Var(&dependencies, "dependency", "explicit dependency model.yaml; repeat")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			io.WriteString(stdout, help)
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "invalid execution arguments; see help")
		return 2
	}
	if *config == "" || runtimeProfile == "" {
		return incomplete("Execution is unavailable without explicit --run-config and --runtime-profile absolute paths.")
	}
	if !filepath.IsAbs(*config) || filepath.Clean(*config) != *config || !filepath.IsAbs(runtimeProfile) || filepath.Clean(runtimeProfile) != runtimeProfile {
		fmt.Fprintln(stderr, "--run-config and --runtime-profile require explicit absolute native paths")
		return 2
	}
	root, err := filepath.Abs(repo)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	selection := project.InputSelection{Repository: root}
	selection.Local, err = selectManifest(root, manifest)
	for _, dependency := range dependencies {
		if err != nil {
			break
		}
		selected, e := selectManifest(root, dependency)
		err = e
		selection.Dependencies = append(selection.Dependencies, selected)
	}
	path, pathErr := snapshot.NewPath(filepath.Base(runtimeProfile))
	if err != nil || pathErr != nil {
		fmt.Fprintln(stderr, "invalid explicit model or Runtime profile selection")
		return 2
	}
	selection.RuntimeProfiles = []project.SourceLocation{{Boundary: filepath.Dir(runtimeProfile), Path: path}}
	report, err := c.apply(ctx, ApplyRequest{Selection: selection, RunConfig: *config, Build: buildinfo.Build()})
	report.FormatVersion, report.Operation = 1, verb
	if err != nil {
		report.Error = err.Error()
	}
	if emit(report) != 0 {
		return 3
	}
	return exitStatus(report.Status, err)
}

func RunOutcome(status model.Status) string {
	if status == model.Passed {
		return "success"
	}
	if status == model.Failed {
		return "failed"
	}
	return "incomplete"
}
