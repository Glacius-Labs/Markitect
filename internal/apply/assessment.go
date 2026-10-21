package apply

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

// entryAssessment names the Model assessment trial's entry in its RunBinding.
const entryAssessment = "model-assessment"

// AssessmentProblemReason is the reason of an assessment whose sealed Pass
// needs correction: Root's integrated result confirmed a material model gap or
// error. The mandatory gate maps it to a Failed Run without realization.
const AssessmentProblemReason = "the model assessment confirmed a material model problem"

// AssessCaptured runs the Model assessment of the captured stand as one
// read-only review Pass of the one Pass process with the caller's purpose,
// mandate and guidance: the evaluation entry of Session C's method trial
// (handoff of 21 October 2026). It binds the same Briefs, ownership, gates and
// budget as Apply in a closed Run: no owner channel, no earlier answers and no
// owner decisions, so missing intent cannot enter behind the model; questions
// stay inside the Manager tree. It runs no required check, because a check
// judges artifacts, not the model, and its RunBinding names the assessment's
// own Pass instead of the strategy's. A Pass that only fresh disputes keep from
// being clean is repeated on the unchanged stand within the Pass budget, as in
// Apply. Nothing is written or published and no Command uses it. The shared
// work volume and the provider homes are removed afterwards.
func AssessCaptured(ctx context.Context, req ApplyRequest, assessment run.PassSpec) (ApplyResult, error) {
	if !assessment.Review || assessment.Writes || assessment.Plans {
		return ApplyResult{}, errors.New("a model assessment is one read-only review Pass")
	}
	cfg, binding, err := applyConfig(req, req.now(), entryAssessment)
	if err != nil {
		return ApplyResult{}, err
	}
	cfg.Checks, cfg.Owner, cfg.Questions, cfg.OwnerDecisions = nil, nil, nil, nil
	binding.Checks, binding.Corrections, binding.Composition = []CheckBinding{}, assessment.Corrections, assessmentDigest(assessment)
	cfg.Binding = binding
	if err := requireVolumeSubpaths(ctx, req.RequireVolumeSubpaths); err != nil {
		return ApplyResult{Binding: binding}, err
	}
	r, err := run.Open(ctx, cfg)
	if err != nil {
		return ApplyResult{Binding: binding}, err
	}
	res := r.Pass(assessment)
	for !res.Stopped && res.Verdict == run.NeedsCorrection && r.Reassess() {
		res = r.Pass(assessment)
	}
	reason := ""
	if !res.Stopped && res.Verdict != run.Satisfied {
		reason = AssessmentProblemReason
	}
	outcome, runErr := r.Finish(reason)
	ports := req.ports()
	ports.removeHomes(ctx, outcome.Homes)
	ports.closeWork(ctx)
	return ApplyResult{Binding: binding, Outcome: outcome}, runErr
}

// assessmentDigest identifies the assessment's Pass declaration, as a
// strategy's digest identifies its Passes.
func assessmentDigest(spec run.PassSpec) string {
	h := sha256.New()
	fmt.Fprintf(h, "%q %q %q %v %v %v %v %v %q\n", spec.Purpose, spec.Mandate, spec.Guidance, spec.Writes, spec.Review, spec.Plans, spec.BindWaits, spec.ArtifactFindingsAdvisory, spec.Corrections)
	return fmt.Sprintf("sha256:%x", h.Sum(nil))
}
