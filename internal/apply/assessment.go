package apply

import (
	"context"
	"errors"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

// AssessCaptured runs one read-only review Pass of the one Pass process on the
// captured stand with the caller's purpose, mandate and guidance: the Model
// assessment experiment of the handoff of 21 October 2026 (Session C), which
// assesses the bound model before any realization. It binds the same Briefs,
// ownership, gates, owner channel and budget as Apply but runs no required
// check, because a check judges artifacts, not the model. A Pass that only
// fresh disputes keep from being clean is repeated on the unchanged stand
// within the Pass budget, as in Apply. Nothing is written or published and no
// Command uses it. The shared work volume and the provider homes are removed
// afterwards.
func AssessCaptured(ctx context.Context, req ApplyRequest, assessment run.PassSpec) (ApplyResult, error) {
	if !assessment.Review || assessment.Writes || assessment.Plans {
		return ApplyResult{}, errors.New("a model assessment is one read-only review Pass")
	}
	cfg, err := applyConfig(req, req.now())
	if err != nil {
		return ApplyResult{}, err
	}
	cfg.Checks = nil
	if err := requireVolumeSubpaths(ctx, req.RequireVolumeSubpaths); err != nil {
		return ApplyResult{}, err
	}
	r, err := run.Open(ctx, cfg)
	if err != nil {
		return ApplyResult{}, err
	}
	res := r.Pass(assessment)
	for !res.Stopped && res.Verdict == run.NeedsCorrection && r.Reassess() {
		res = r.Pass(assessment)
	}
	reason := ""
	if !res.Stopped && res.Verdict != run.Satisfied {
		reason = "the model assessment left genuine findings open"
	}
	outcome, runErr := r.Finish(reason)
	ports := req.ports()
	ports.removeHomes(ctx, outcome.Homes)
	ports.closeWork(ctx)
	return ApplyResult{Outcome: outcome}, runErr
}
