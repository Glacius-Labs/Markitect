package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ApplyRequest contains parsed invocation coordinates, never credentials,
// resolved profiles or a preview publication guard. Build is the
// implementation identity of this product build, which the Run binds.
type ApplyRequest struct {
	Selection apply.InputSelection
	RunConfig string
	Build     string
}

// ApplyReport projects preparation, the assignment lifecycle, publication and
// retained evidence. Run is retained for historical report consumers.
// ReportValue derives its JSON outcome from Status for both stdout and retained
// evidence: success only for a clean Verify with an applied or no-change
// publication and read-back, failed for a definite rejection, incomplete otherwise.
type ApplyReport struct {
	FormatVersion     int                       `json:"formatVersion"`
	Operation         string                    `json:"operation"`
	Status            model.Status              `json:"status"`
	Preparation       any                       `json:"preparation,omitempty"`
	Binding           *apply.RunBinding         `json:"binding,omitempty"`
	Run               *RunReport                `json:"run,omitempty"`
	Assignment        *AssignmentReport         `json:"assignment,omitempty"`
	Publication       *apply.PublicationReceipt `json:"publication,omitempty"`
	RetainedWork      string                    `json:"retainedWork,omitempty"`
	EvidenceDirectory string                    `json:"evidenceDirectory,omitempty"`
	Error             string                    `json:"error,omitempty"`
}

// AssignmentReport names the observed lifecycle without projecting it as
// legacy Passes or claiming publication from a provider completion event.
type AssignmentReport struct {
	ID                string                   `json:"id"`
	ModelHead         string                   `json:"modelHead,omitempty"`
	AssessmentCommit  string                   `json:"assessmentCommit,omitempty"`
	RootCommit        string                   `json:"rootCommit,omitempty"`
	AcceptedCommit    string                   `json:"acceptedCommit,omitempty"`
	ReviewCount       int                      `json:"reviewCount"`
	CheckCount        int                      `json:"checkCount"`
	Messages          int                      `json:"messages"`
	NoChange          bool                     `json:"noChange"`
	ModelProblem      bool                     `json:"modelProblem,omitempty"`
	ModelProblemIssue *run.ReviewIssue         `json:"modelProblemIssue,omitempty"`
	Assessment        AssignmentReviewReport   `json:"assessment"`
	Homes             []string                 `json:"homes"`
	HomesRemoved      []string                 `json:"homesRemoved"`
	GitContainerID    string                   `json:"gitContainerID,omitempty"`
	GitExecCount      int                      `json:"gitExecCount"`
	Reviews           []AssignmentReviewReport `json:"reviews"`
	Checks            []AssignmentCheckReport  `json:"checks"`
}

type AssignmentReviewReport struct {
	AcceptedCommit string            `json:"acceptedCommit,omitempty"`
	Reason         string            `json:"reason,omitempty"`
	ModelProblem   bool              `json:"modelProblem,omitempty"`
	Issues         []run.ReviewIssue `json:"issues"`
}

type AssignmentCheckReport struct {
	Commit  string           `json:"commit"`
	Name    string           `json:"name"`
	Passed  bool             `json:"passed"`
	Error   string           `json:"error,omitempty"`
	Receipt run.CheckReceipt `json:"receipt"`
}

func NewAssignmentReport(r apply.AssignmentRealizationResult) *AssignmentReport {
	out := &AssignmentReport{ID: r.ID, ModelHead: r.ModelHead, AssessmentCommit: r.Assessment.AcceptedCommit, RootCommit: r.Root.ResultCommit, AcceptedCommit: r.AcceptedCommit, ReviewCount: len(r.Reviews), CheckCount: len(r.Checks), Messages: r.Messages, NoChange: r.NoChange, ModelProblem: r.ModelProblem != nil, ModelProblemIssue: r.ModelProblem, Reviews: []AssignmentReviewReport{}, Checks: []AssignmentCheckReport{}}
	out.Assessment = AssignmentReviewReport{AcceptedCommit: r.Assessment.AcceptedCommit, Reason: r.Assessment.Reason, ModelProblem: r.Assessment.ModelProblem, Issues: append([]run.ReviewIssue{}, r.Assessment.Issues...)}
	out.Homes = []string{}
	out.HomesRemoved = append([]string{}, r.HomesRemoved...)
	seen := map[string]bool{}
	addHome := func(name string) {
		if name != "" && !seen[name] {
			seen[name] = true
			out.Homes = append(out.Homes, name)
		}
	}
	addHome(r.Assessment.Home)
	if r.Work != nil {
		for _, name := range r.Work.Homes() {
			addHome(name)
		}
	}
	for _, review := range r.Reviews {
		addHome(review.Home)
	}
	for _, review := range r.Reviews {
		out.Reviews = append(out.Reviews, AssignmentReviewReport{AcceptedCommit: review.AcceptedCommit, Reason: review.Reason, ModelProblem: review.ModelProblem, Issues: append([]run.ReviewIssue{}, review.Issues...)})
	}
	for _, check := range r.Checks {
		out.Checks = append(out.Checks, AssignmentCheckReport{Commit: check.Commit, Name: check.Name, Passed: check.Error == "" && check.Receipt.Passed(check.Request), Error: check.Error, Receipt: check.Receipt})
	}
	return out
}

func AssignmentApplyStatus(r apply.AssignmentRealizationResult, receipt *apply.PublicationReceipt, err error) model.Status {
	if r.ModelProblem != nil || receipt != nil && receipt.Outcome == apply.PublicationFailed {
		return model.Failed
	}
	if err != nil || r.AcceptedCommit == "" || receipt == nil {
		return model.NotCompleted
	}
	if receipt.Outcome == apply.PublicationApplied || receipt.Outcome == apply.PublicationNoChange {
		return model.Passed
	}
	return model.NotCompleted
}

// RunReport is the run's outcome without the candidate's bytes.
type RunReport struct {
	ID        string             `json:"id"`
	Status    run.Status         `json:"status"`
	Reason    string             `json:"reason,omitempty"`
	Passes    []run.PassRecord   `json:"passes"`
	Findings  []run.Finding      `json:"findings"`
	Questions []run.Question     `json:"questions"`
	Problems  []run.Problem      `json:"problems"`
	Turns     []run.TurnRecord   `json:"turns"`
	Probes    []run.ProbeReceipt `json:"probes"`
	Usage     run.Usage          `json:"usage"`
}

// NewRunReport projects an outcome. Lists are never null.
func NewRunReport(id string, outcome run.Outcome) *RunReport {
	r := &RunReport{ID: id, Status: outcome.Status, Reason: outcome.Reason, Passes: outcome.Passes, Findings: outcome.Findings, Questions: outcome.Questions, Problems: outcome.Problems, Turns: outcome.Turns, Probes: outcome.Probes, Usage: outcome.Usage}
	if r.Probes == nil {
		r.Probes = []run.ProbeReceipt{}
	}
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

type ApplyFunc func(context.Context, ApplyRequest) (ApplyReport, error)

func (c *Command) execution(ctx context.Context, verb string, args []string, stdout, stderr io.Writer, emit func(any) int) int {
	incomplete := func(reason string) int {
		if emit(map[string]string{"operation": verb, "status": "not_completed", "reason": reason}) != 0 {
			return 3
		}
		return 3
	}
	if verb == "apply" && c.apply == nil || verb == "extract" && c.extract == nil {
		return incomplete("Execution is unavailable because its concrete execution/publication ports are not configured.")
	}
	flags := flag.NewFlagSet(verb, flag.ContinueOnError)
	flags.SetOutput(stderr)
	config := flags.String("run-config", "", "explicit absolute invocation configuration")
	var repo, manifest, runtimeProfile string
	var dependencies manifestList
	flags.StringVar(&repo, "repo", ".", "repository root")
	if verb == "apply" {
		flags.StringVar(&manifest, "model", ".markitect/model.yaml", "explicit model.yaml")
	} else {
		manifest = extractManifest
	}
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
