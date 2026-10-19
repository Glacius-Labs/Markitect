package cli

import (
	"context"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// extractManifest is the Host-prepared structural manifest of an Extract; the
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
	r := ExtractReport{Deviations: []apply.ArtifactDeviation{}, ProposedChecks: []apply.ProposedCheck{}, OpenQuestions: []run.Question{}, Publication: result.Receipt, RetainedWork: result.RetainedWork}
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
