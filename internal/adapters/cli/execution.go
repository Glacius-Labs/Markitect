package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ExecutionRequest contains parsed invocation coordinates, never credentials,
// resolved profiles, invented lifecycle records or a preview publication guard.
type ExecutionRequest struct {
	Selection        apply.InputSelection
	RunConfig        string
	Question, Answer string
}

// ExecutionReport projects actual preparation/control/execution observations.
// OpenQuestions retains the producer's concrete owner-question values. The CLI
// reports them without assigning lifecycle or publication meaning of its own.
type ExecutionReport struct {
	FormatVersion int                    `json:"formatVersion"`
	Operation     string                 `json:"operation"`
	Status        model.Status           `json:"status"`
	Preparation   any                    `json:"preparation,omitempty"`
	Control       apply.ControlBinding   `json:"control"`
	Execution     *apply.ExecutionResult `json:"execution,omitempty"`
	OpenQuestions any                    `json:"openQuestions,omitempty"`
	Error         string                 `json:"error,omitempty"`
}

type ExecuteFunc func(context.Context, ExecutionRequest) (ExecutionReport, error)

func (c *Command) execution(ctx context.Context, verb string, args []string, stdout, stderr io.Writer, emit func(any) int) int {
	incomplete := func(reason string) int {
		if emit(map[string]string{"operation": verb, "status": "not_completed", "reason": reason}) != 0 {
			return 3
		}
		return 3
	}
	if c.execute == nil {
		return incomplete("Execution is unavailable because its concrete execution/publication ports are not configured.")
	}
	question := ""
	if verb == "answer" {
		if len(args) == 0 || args[0] == "" || args[0][0] == '-' {
			fmt.Fprintln(stderr, "answer requires one open owner-question identity")
			return 2
		}
		question, args = args[0], args[1:]
	}
	flags := flag.NewFlagSet(verb, flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "repository root")
	manifest := flags.String("model", ".markitect/model.yaml", "explicit model.yaml")
	host := flags.String("host-profile", "", "explicit absolute Host profile document")
	config := flags.String("run-config", "", "explicit absolute invocation configuration")
	var answer string
	if verb == "answer" {
		flags.StringVar(&answer, "text", "", "owner answer text")
	}
	var dependencies manifestList
	flags.Var(&dependencies, "dependency", "explicit dependency model.yaml; repeat")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			io.WriteString(stdout, help)
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || verb == "answer" && answer == "" {
		fmt.Fprintln(stderr, "invalid execution arguments; see help")
		return 2
	}
	if *config == "" || *host == "" {
		return incomplete("Execution is unavailable without explicit --run-config and --host-profile absolute paths.")
	}
	if !filepath.IsAbs(*config) || filepath.Clean(*config) != *config || !filepath.IsAbs(*host) || filepath.Clean(*host) != *host {
		fmt.Fprintln(stderr, "--run-config and --host-profile require explicit absolute native paths")
		return 2
	}
	root, err := filepath.Abs(*repo)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	selection := apply.InputSelection{Repository: root}
	selection.Local, err = selectManifest(root, *manifest)
	for _, dependency := range dependencies {
		if err != nil {
			break
		}
		selected, e := selectManifest(root, dependency)
		err = e
		selection.Dependencies = append(selection.Dependencies, selected)
	}
	path, pathErr := snapshot.NewPath(filepath.Base(*host))
	if err != nil || pathErr != nil {
		fmt.Fprintln(stderr, "invalid explicit model or Host selection")
		return 2
	}
	selection.HostProfiles = []apply.SourceLocation{{Boundary: filepath.Dir(*host), Path: path}}
	report, err := c.execute(ctx, ExecutionRequest{Selection: selection, RunConfig: *config, Question: question, Answer: answer})
	report.FormatVersion, report.Operation = 1, verb
	if err != nil {
		report.Error = err.Error()
	}
	if emit(report) != 0 {
		return 3
	}
	return exitStatus(report.Status, err)
}
