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
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ApplyRequest contains parsed invocation coordinates, never credentials,
// resolved profiles or a preview publication guard.
type ApplyRequest struct {
	Selection apply.InputSelection
	RunConfig string
}

// AnswerRequest carries the answer of the invoking AI to one open question of
// a running Apply or Extract. Run binds it to that exact run; RunConfig selects
// the state directory the run uses.
type AnswerRequest struct {
	Question  string
	Run       string
	Text      string
	RunConfig string
}

// ApplyReport projects the observations of one Apply: the preparation, the
// run with its Passes, the publication and where the evidence is kept. Outcome
// is the Run outcome: success only for a clean Verify with an applied or
// no-change publication and read-back, failed for a definite rejection,
// incomplete otherwise.
type ApplyReport struct {
	FormatVersion     int                       `json:"formatVersion"`
	Operation         string                    `json:"operation"`
	Status            model.Status              `json:"status"`
	Outcome           string                    `json:"outcome,omitempty"`
	Preparation       any                       `json:"preparation,omitempty"`
	Run               *RunReport                `json:"run,omitempty"`
	Publication       *apply.PublicationReceipt `json:"publication,omitempty"`
	RetainedWork      string                    `json:"retainedWork,omitempty"`
	EvidenceDirectory string                    `json:"evidenceDirectory,omitempty"`
	Error             string                    `json:"error,omitempty"`
}

// RunReport is the run's outcome without the candidate's bytes.
type RunReport struct {
	ID        string           `json:"id"`
	Status    run.Status       `json:"status"`
	Reason    string           `json:"reason,omitempty"`
	Passes    []run.PassRecord `json:"passes"`
	Findings  []run.Finding    `json:"findings"`
	Questions []run.Question   `json:"questions"`
	Problems  []run.Problem    `json:"problems"`
	Turns     []run.TurnRecord `json:"turns"`
	Usage     run.Usage        `json:"usage"`
}

// NewRunReport projects an outcome. Lists are never null.
func NewRunReport(id string, outcome run.Outcome) *RunReport {
	r := &RunReport{ID: id, Status: outcome.Status, Reason: outcome.Reason, Passes: outcome.Passes, Findings: outcome.Findings, Questions: outcome.Questions, Problems: outcome.Problems, Turns: outcome.Turns, Usage: outcome.Usage}
	if r.Passes == nil {
		r.Passes = []run.PassRecord{}
	}
	if r.Problems == nil {
		r.Problems = []run.Problem{}
	}
	if r.Findings == nil {
		r.Findings = []run.Finding{}
	}
	if r.Questions == nil {
		r.Questions = []run.Question{}
	}
	if r.Turns == nil {
		r.Turns = []run.TurnRecord{}
	}
	return r
}

// ApplyStatus is Passed only for a verified run whose candidate was published
// as applied or no change, without an error. A failed run is Failed even though
// it comes with an error, and so is a definitely failed publication; everything
// else is not completed.
func ApplyStatus(outcome run.Outcome, receipt *apply.PublicationReceipt, err error) model.Status {
	if outcome.Status == run.Failed {
		return model.Failed
	}
	if err != nil {
		return model.NotCompleted
	}
	switch outcome.Status {
	case run.Verified:
		if receipt == nil {
			return model.NotCompleted
		}
		switch receipt.Outcome {
		case apply.PublicationApplied, apply.PublicationNoChange:
			return model.Passed
		case apply.PublicationFailed:
			return model.Failed
		}
	}
	return model.NotCompleted
}

// RunOutcome names the Run outcome of a command status: success, failed or
// incomplete.
func RunOutcome(status model.Status) string {
	switch status {
	case model.Passed:
		return "success"
	case model.Failed:
		return "failed"
	}
	return "incomplete"
}

// AnswerReport states what the answer command wrote.
type AnswerReport struct {
	FormatVersion int          `json:"formatVersion"`
	Operation     string       `json:"operation"`
	Status        model.Status `json:"status"`
	Question      string       `json:"question"`
	Path          string       `json:"path,omitempty"`
	Bytes         int          `json:"bytes,omitempty"`
	Error         string       `json:"error,omitempty"`
}

type ApplyFunc func(context.Context, ApplyRequest) (ApplyReport, error)
type AnswerFunc func(context.Context, AnswerRequest) (AnswerReport, error)

func (c *Command) execution(ctx context.Context, verb string, args []string, stdout, stderr io.Writer, emit func(any) int) int {
	incomplete := func(reason string) int {
		if emit(map[string]string{"operation": verb, "status": "not_completed", "reason": reason}) != 0 {
			return 3
		}
		return 3
	}
	if verb == "apply" && c.apply == nil || verb == "answer" && c.answer == nil || verb == "extract" && c.extract == nil {
		return incomplete("Execution is unavailable because its concrete execution/publication ports are not configured.")
	}
	question := ""
	if verb == "answer" {
		if len(args) == 0 || args[0] == "" || args[0][0] == '-' {
			fmt.Fprintln(stderr, "answer requires one open question identity")
			return 2
		}
		question, args = args[0], args[1:]
	}
	var runID string
	flags := flag.NewFlagSet(verb, flag.ContinueOnError)
	flags.SetOutput(stderr)
	config := flags.String("run-config", "", "explicit absolute invocation configuration")
	var repo, manifest, runtimeProfile, text string
	var dependencies manifestList
	if verb == "apply" || verb == "extract" {
		flags.StringVar(&repo, "repo", ".", "repository root")
		if verb == "apply" {
			flags.StringVar(&manifest, "model", ".markitect/model.yaml", "explicit model.yaml")
		} else {
			manifest = extractManifest
		}
		flags.StringVar(&runtimeProfile, "runtime-profile", "", "explicit absolute Runtime profile document")
		flags.Var(&dependencies, "dependency", "explicit dependency model.yaml; repeat")
	} else {
		flags.StringVar(&text, "text", "", "answer text")
		flags.StringVar(&runID, "run", "", "identity of the run that asked")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			io.WriteString(stdout, help)
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || verb == "answer" && (text == "" || runID == "") {
		fmt.Fprintln(stderr, "invalid execution arguments; see help")
		return 2
	}
	if *config == "" || verb != "answer" && runtimeProfile == "" {
		return incomplete("Execution is unavailable without explicit --run-config and --runtime-profile absolute paths.")
	}
	if !filepath.IsAbs(*config) || filepath.Clean(*config) != *config || verb != "answer" && (!filepath.IsAbs(runtimeProfile) || filepath.Clean(runtimeProfile) != runtimeProfile) {
		fmt.Fprintln(stderr, "--run-config and --runtime-profile require explicit absolute native paths")
		return 2
	}
	if verb == "answer" {
		report, err := c.answer(ctx, AnswerRequest{Question: question, Run: runID, Text: text, RunConfig: *config})
		report.FormatVersion, report.Operation, report.Question = 1, verb, question
		if err != nil {
			report.Error = err.Error()
		}
		if emit(report) != 0 {
			return 3
		}
		return exitStatus(report.Status, err)
	}
	root, err := filepath.Abs(repo)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	selection := apply.InputSelection{Repository: root}
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
	selection.RuntimeProfiles = []apply.SourceLocation{{Boundary: filepath.Dir(runtimeProfile), Path: path}}
	if verb == "extract" {
		report, err := c.extract(ctx, ExtractRequest{Selection: selection, RunConfig: *config})
		report.FormatVersion, report.Operation = 1, verb
		if err != nil {
			report.Error = err.Error()
		}
		if emit(report) != 0 {
			return 3
		}
		// The account is the last thing the command writes to stderr.
		io.WriteString(stderr, RenderExtractAccount(report))
		return exitStatus(report.Status, err)
	}
	report, err := c.apply(ctx, ApplyRequest{Selection: selection, RunConfig: *config})
	report.FormatVersion, report.Operation = 1, verb
	if err != nil {
		report.Error = err.Error()
	}
	report.Outcome = RunOutcome(report.Status)
	if emit(report) != 0 {
		return 3
	}
	return exitStatus(report.Status, err)
}
