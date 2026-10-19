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

// AnswerRequest carries the owner's answer to one open question of a running
// Apply. RunConfig selects the state directory that Apply uses.
type AnswerRequest struct {
	Question  string
	Text      string
	RunConfig string
}

// ApplyReport projects the observations of one Apply: the preparation, the
// reconciliation run, the publication and where the evidence is kept.
type ApplyReport struct {
	FormatVersion     int                       `json:"formatVersion"`
	Operation         string                    `json:"operation"`
	Status            model.Status              `json:"status"`
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
	Rounds    []run.Round      `json:"rounds"`
	Findings  []run.Finding    `json:"findings"`
	Questions []run.Question   `json:"questions"`
	Turns     []run.TurnRecord `json:"turns"`
	Usage     run.Usage        `json:"usage"`
}

// NewRunReport projects an outcome. Lists are never null.
func NewRunReport(id string, outcome run.Outcome) *RunReport {
	r := &RunReport{ID: id, Status: outcome.Status, Reason: outcome.Reason, Rounds: outcome.Rounds, Findings: outcome.Findings, Questions: outcome.Questions, Turns: outcome.Turns, Usage: outcome.Usage}
	if r.Rounds == nil {
		r.Rounds = []run.Round{}
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
// as applied or no change, without an error. A failed run or a definitely
// failed publication is Failed; everything else is not completed.
func ApplyStatus(outcome run.Outcome, receipt *apply.PublicationReceipt, err error) model.Status {
	if err != nil {
		return model.NotCompleted
	}
	switch outcome.Status {
	case run.Failed:
		return model.Failed
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
	if verb == "apply" && c.apply == nil || verb == "answer" && c.answer == nil {
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
	config := flags.String("run-config", "", "explicit absolute invocation configuration")
	var repo, manifest, host, text string
	var dependencies manifestList
	if verb == "apply" {
		flags.StringVar(&repo, "repo", ".", "repository root")
		flags.StringVar(&manifest, "model", ".markitect/model.yaml", "explicit model.yaml")
		flags.StringVar(&host, "host-profile", "", "explicit absolute Host profile document")
		flags.Var(&dependencies, "dependency", "explicit dependency model.yaml; repeat")
	} else {
		flags.StringVar(&text, "text", "", "owner answer text")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			io.WriteString(stdout, help)
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || verb == "answer" && text == "" {
		fmt.Fprintln(stderr, "invalid execution arguments; see help")
		return 2
	}
	if *config == "" || verb == "apply" && host == "" {
		return incomplete("Execution is unavailable without explicit --run-config and --host-profile absolute paths.")
	}
	if !filepath.IsAbs(*config) || filepath.Clean(*config) != *config || verb == "apply" && (!filepath.IsAbs(host) || filepath.Clean(host) != host) {
		fmt.Fprintln(stderr, "--run-config and --host-profile require explicit absolute native paths")
		return 2
	}
	if verb == "answer" {
		report, err := c.answer(ctx, AnswerRequest{Question: question, Text: text, RunConfig: *config})
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
	path, pathErr := snapshot.NewPath(filepath.Base(host))
	if err != nil || pathErr != nil {
		fmt.Fprintln(stderr, "invalid explicit model or Host selection")
		return 2
	}
	selection.HostProfiles = []apply.SourceLocation{{Boundary: filepath.Dir(host), Path: path}}
	report, err := c.apply(ctx, ApplyRequest{Selection: selection, RunConfig: *config})
	report.FormatVersion, report.Operation = 1, verb
	if err != nil {
		report.Error = err.Error()
	}
	if emit(report) != 0 {
		return 3
	}
	return exitStatus(report.Status, err)
}
