package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// extractManifest is the Orchestrator-prepared structural manifest of an Extract; the
// command has no --model flag.
const extractManifest = ".markitect/model.yaml"

// ExtractRequest contains parsed invocation coordinates, never credentials,
// resolved profiles or a preview publication guard.
type ExtractRequest struct {
	Selection apply.InputSelection
	RunConfig string
}

// ExtractReport projects the observations of one Extract: the preparation, the
// two phases, the model's completion, the final Verify result, the artifact
// deviations per Definition, the proposed checks and open questions, the
// publication and where the evidence is kept.
type ExtractReport struct {
	FormatVersion int          `json:"formatVersion"`
	Operation     string       `json:"operation"`
	Status        model.Status `json:"status"`
	Preparation   any          `json:"preparation,omitempty"`
	// ModelComplete is true only when the second phase verified the model.
	ModelComplete     bool                      `json:"modelComplete"`
	Structure         *RunReport                `json:"structure,omitempty"`
	Model             *RunReport                `json:"model,omitempty"`
	Verify            *VerifyReport             `json:"verify,omitempty"`
	Deviations        []apply.ArtifactDeviation `json:"deviations"`
	ProposedChecks    []apply.ProposedCheck     `json:"proposedChecks"`
	OpenQuestions     []run.Question            `json:"openQuestions"`
	Publication       *apply.PublicationReceipt `json:"publication,omitempty"`
	RetainedWork      string                    `json:"retainedWork,omitempty"`
	EvidenceDirectory string                    `json:"evidenceDirectory,omitempty"`
	Error             string                    `json:"error,omitempty"`
	// Packages are the Packages of a verified model. They only serve the readable
	// account, so the JSON report does not carry them.
	Packages []apply.ExtractedPackage `json:"-"`
}

// VerifyReport is the last Verify round of the model phase.
type VerifyReport struct {
	Round       int              `json:"round"`
	Green       bool             `json:"green"`
	Assessments []run.Assessment `json:"assessments"`
}

// NewExtractReport projects the result of an Extract with the run identity. Lists
// are never null.
func NewExtractReport(id string, result apply.ExtractResult) ExtractReport {
	r := ExtractReport{Deviations: []apply.ArtifactDeviation{}, ProposedChecks: []apply.ProposedCheck{}, OpenQuestions: []run.Question{}, Publication: result.Receipt, RetainedWork: result.RetainedWork, Packages: result.Packages}
	questions := result.Structure.Questions
	if result.Structure.Status != "" {
		r.Structure = NewRunReport(id+"-"+apply.ExtractStructurePhase, result.Structure)
	}
	if result.Model != nil {
		r.Model = NewRunReport(id+"-"+apply.ExtractModelPhase, *result.Model)
		// The model phase continues the structure phase's owner channel, so its
		// questions include the structure phase's; a model phase that failed to
		// start has none.
		if len(result.Model.Questions) >= len(questions) {
			questions = result.Model.Questions
		}
		r.ModelComplete = result.Model.Status == run.Verified
		r.Deviations = append(r.Deviations, apply.ExtractDeviations(result.Model.Findings)...)
		if rounds := result.Model.Rounds; len(rounds) > 0 {
			last := rounds[len(rounds)-1]
			r.Verify = &VerifyReport{Round: last.Number, Green: last.Green, Assessments: last.Assessments}
		}
	}
	r.ProposedChecks = append(r.ProposedChecks, result.ProposedChecks...)
	for _, q := range questions {
		if !q.Answered {
			r.OpenQuestions = append(r.OpenQuestions, q)
		}
	}
	return r
}

// ExtractStatus is Passed only for a verified model that was published as
// applied or no change, without an error. A phase that failed definitely and a
// definitely failed publication are Failed; everything else is not completed.
func ExtractStatus(result apply.ExtractResult, err error) model.Status {
	if result.Structure.Status == run.Failed || result.Model != nil && result.Model.Status == run.Failed {
		return model.Failed
	}
	if err != nil || result.Model == nil {
		return model.NotCompleted
	}
	return ApplyStatus(*result.Model, result.Receipt, nil)
}

type ExtractFunc func(context.Context, ExtractRequest) (ExtractReport, error)

// Limits of the readable account: it stays a short summary, so long lists end
// with "and N more" and long texts are shortened.
const (
	accountObservedRunes = 120
	accountTextRunes     = 100
	accountReasonRunes   = 240
	accountMaxPackages   = 12
	accountMaxQuestions  = 6
	accountMaxDeviations = 5
	accountMaxChecks     = 5
	accountMaxManagers   = 8
)

// RenderExtractAccount is the owner's short readable account of an Extract:
// status, Packages, owner questions, deviations, proposed checks, Verify,
// publication and, for a published model, the next step. It is plain text for
// the terminal and is rendered from the report alone.
func RenderExtractAccount(r ExtractReport) string {
	var b strings.Builder
	line := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }
	ran := r.Model != nil

	line("%s", extractStatusLine(r))

	if len(r.Packages) > 0 {
		shown := r.Packages[:min(len(r.Packages), accountMaxPackages)]
		line("")
		line("Packages (%d)", len(r.Packages))
		width := 0
		for _, p := range shown {
			width = max(width, len(p.Path))
		}
		for _, p := range shown {
			if p.Title == "" {
				line("  %s", p.Path)
			} else {
				line("  %-*s  %s", width, p.Path, shorten(p.Title, accountTextRunes))
			}
		}
		more(&b, len(r.Packages), len(shown))
	}

	questions := accountQuestions(r)
	if len(questions) > 0 {
		shown := questions[:min(len(questions), accountMaxQuestions)]
		line("")
		line("Owner questions (%d)", len(questions))
		for _, q := range shown {
			answer := "unanswered"
			if q.Answered {
				answer = shorten(q.Answer, accountTextRunes)
			}
			line("  %s %s -> %s", q.ID, shorten(q.Text, accountTextRunes), answer)
		}
		more(&b, len(questions), len(shown))
	} else if ran {
		line("")
		line("Owner questions: none")
	}

	if len(r.Deviations) > 0 {
		shown := r.Deviations[:min(len(r.Deviations), accountMaxDeviations)]
		line("")
		line("Realization deviates from the model (%d)", len(r.Deviations))
		for _, d := range shown {
			observed := ""
			if len(d.Observed) > 0 {
				observed = d.Observed[0]
			}
			line("  %s (owners: %s)", d.Definition, strings.Join(d.Owners, ", "))
			line("    %s", shorten(observed, accountObservedRunes))
		}
		more(&b, len(r.Deviations), len(shown))
	} else if ran {
		line("")
		line("Realization deviations: none")
	}

	if len(r.ProposedChecks) > 0 {
		shown := r.ProposedChecks[:min(len(r.ProposedChecks), accountMaxChecks)]
		line("")
		line("Proposed checks (%d)", len(r.ProposedChecks))
		for _, c := range shown {
			line("  %s: %s in %s", c.ID, shorten(strings.Join(c.Args, " "), accountTextRunes), c.Directory)
		}
		more(&b, len(r.ProposedChecks), len(shown))
	}

	line("")
	if r.Verify == nil {
		line("Verify: no round finished")
	} else {
		var satisfied, open []string
		for _, a := range r.Verify.Assessments {
			if a.Satisfied {
				satisfied = append(satisfied, a.Manager)
			} else {
				open = append(open, a.Manager)
			}
		}
		line("Verify (round %d)", r.Verify.Round)
		if len(satisfied) > 0 {
			line("  satisfied: %s", managerList(satisfied))
		}
		if len(open) > 0 {
			line("  not satisfied: %s", managerList(open))
		}
		line("  open advisory findings: %d", openAdvisoryFindings(r))
	}

	line("")
	line("Publication: %s", publicationText(r))
	if r.RetainedWork != "" {
		line("Retained work: %s (kept for inspection)", r.RetainedWork)
	}
	if r.EvidenceDirectory != "" {
		line("Evidence: %s", r.EvidenceDirectory)
	}

	if r.Status == model.Passed && (r.Publication == nil || r.Publication.Outcome != apply.PublicationNoChange) {
		line("")
		line("Next: review and commit .markitect/model, .markitect/registry and")
		line("  .markitect/execution.yaml to adopt the model. Extract commits nothing.")
	}
	return b.String()
}

// extractStatusLine is the first line of the account: complete and published,
// incomplete with its reason, or failed.
func extractStatusLine(r ExtractReport) string {
	switch {
	case r.Status == model.Passed && r.Publication != nil && r.Publication.Outcome == apply.PublicationNoChange:
		return "Model complete; the repository already holds it."
	case r.Status == model.Passed:
		return "Model complete and published."
	case r.Status == model.Failed:
		return "Extract failed: " + extractReason(r) + "."
	}
	return "Model incomplete: " + extractReason(r) + "."
}

// extractReason says why an Extract did not finish: the publication of a
// verified model, else the model phase's, else the structure phase's reason,
// then the reported error.
func extractReason(r ExtractReport) string {
	var parts []string
	switch {
	case r.ModelComplete:
		outcome := "no publication"
		if r.Publication != nil {
			outcome = string(r.Publication.Outcome)
			if r.Publication.Reason != "" {
				outcome += ": " + r.Publication.Reason
			}
		}
		parts = append(parts, "the verified model was not published ("+outcome+")")
	case r.Model != nil && r.Model.Reason != "":
		parts = append(parts, r.Model.Reason)
	case r.Structure != nil && r.Structure.Reason != "":
		parts = append(parts, "structure phase: "+r.Structure.Reason)
	}
	if r.Error != "" && !strings.Contains(strings.Join(parts, " "), r.Error) {
		parts = append(parts, r.Error)
	}
	if len(parts) == 0 {
		return "no reason reported"
	}
	return strings.TrimRight(shorten(strings.Join(parts, "; "), accountReasonRunes), ".")
}

// accountQuestions are the questions of the final phase, answered or not: the
// model phase continues the structure phase's channel, so its list includes
// the structure phase's.
func accountQuestions(r ExtractReport) []run.Question {
	switch {
	case r.Model != nil && (r.Structure == nil || len(r.Model.Questions) >= len(r.Structure.Questions)):
		return r.Model.Questions
	case r.Structure != nil:
		return r.Structure.Questions
	}
	return nil
}

// openAdvisoryFindings counts the open findings of the model phase that do not
// block and are not artifact deviations, which the account lists apart.
func openAdvisoryFindings(r ExtractReport) int {
	n := 0
	if r.Model != nil {
		for _, f := range r.Model.Findings {
			if !f.Closed && !f.Blocking && f.Subject != "artifact" {
				n++
			}
		}
	}
	return n
}

// publicationText states the publication outcome and how many files it changed.
func publicationText(r ExtractReport) string {
	p := r.Publication
	if p == nil {
		return "none"
	}
	switch p.Outcome {
	case apply.PublicationApplied:
		if len(p.Paths) == 1 {
			return "applied, 1 file changed"
		}
		return fmt.Sprintf("applied, %d files changed", len(p.Paths))
	case apply.PublicationNoChange:
		return "no change, nothing to write"
	}
	written := 0
	for _, path := range p.Paths {
		if path.State == apply.PathAfter {
			written++
		}
	}
	text := fmt.Sprintf("%s, %d of %d files written", p.Outcome, written, len(p.Paths))
	if p.Reason != "" {
		text += ": " + shorten(p.Reason, accountTextRunes)
	}
	return text
}

// managerList joins Manager paths, ending a long list with "and N more".
func managerList(managers []string) string {
	shown := managers[:min(len(managers), accountMaxManagers)]
	text := strings.Join(shown, ", ")
	if len(managers) > len(shown) {
		text += fmt.Sprintf(" and %d more", len(managers)-len(shown))
	}
	return text
}

// more ends a capped list.
func more(b *strings.Builder, total, shown int) {
	if total > shown {
		fmt.Fprintf(b, "  and %d more\n", total-shown)
	}
}

// shorten collapses whitespace and cuts the text to at most n characters, with
// an ellipsis when it was longer.
func shorten(text string, n int) string {
	text = strings.Join(strings.Fields(text), " ")
	if runes := []rune(text); len(runes) > n {
		return string(runes[:n-3]) + "..."
	}
	return text
}
