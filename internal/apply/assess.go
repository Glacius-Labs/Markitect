package apply

import (
	"context"

	"github.com/Glacius-Labs/Markitect/internal/run"
)

// The mandatory Model assessment of a model-to-artifact workflow (owner
// decision of 21 October 2026): after the strict Model check and File check
// and before any work Pass, every Manager assesses the bound model as a fresh
// read-only reviewer of the same recursive Pass, parents first, and Root
// integrates. An accepted assessment releases the work; a material problem
// Root confirms ends the run Failed, because nothing in the run can correct
// the model: it is corrected outside and a new run starts. The mandate and the
// guidance are Session C's method (experiments/model-assessment/assessment-pass.json at 7db130d7); a finding's
// correction carries its class.

const assessMandate = "This Pass is a Model assessment: you assess the bound model itself, before any realization, and you change nothing. The question is whether the model is usable, materially consistent and decisive enough for its realization, not whether it is as detailed as possible or follows the architecture you would prefer. The candidate is the model with its linked and included Definitions; artifacts in the stand are not assessed, are no evidence of what the model intends and never settle what it leaves open. Nobody outside the team answers during this Run and the model cannot change within it: a material decision the model lacks is a model problem, never a question to the user, a default or a value you choose. Assess each of your own obligations against four questions. First, public decisions: is every result, acceptance or refusal and other outcome that users or other areas rely on stated directly or through a linked or included Definition, derivable from them, or explicitly delegated as implementation freedom? Second, consistency: does a statement contradict another of your own, a linked or an included Definition? Third, responsibility and contracts: are its responsibility and the contracts it needs from or offers to other areas usable to realize and verify it independently? Fourth, freedom: which choices does it deliberately leave to the implementer? That freedom, details the model leaves out because they are ordinary implementation choices, and an unusual or unfashionable design that the model states consistently are no issue. Read the complete statements before you judge, since a linked or included Definition often settles an apparent gap. A concrete suspected problem is investigated before your result: look up the linked Definitions, ask your parent or a child inside the team, and then classify it as a model problem, resolve it with the model text, or keep working; never report satisfied while a relevant suspicion is unresolved, and never park it as a note. Cover every own obligation in your result: done when you assessed it, with at least one evidence item that cites the inspected model file as source, the Definition and statement as location and your reasoning. Report each model problem as a genuine finding, with correction model_gap for a missing material decision or an unusable contract and model_error for a contradiction or an erroneous statement; always state the correction. An optional idea is at most a note with blocking false. There is no quota: a model without a model problem is accepted. Never invent a requirement, demand detail the model may leave out, infer intended policy from existing code or common conventions, or require an application to exist. A finding that was resolved or disproved is not raised again on the same statements without new evidence. As a parent, also assess the interfaces between your children and between them and you, judge each child's findings and their classification against the statements they cite, dispute a finding only when the model already settles it, leaves it to ordinary implementation or grants it as freedom, return a result whose finding is real but names the wrong class so that the child corrects the class, and return a result whose acceptance its evidence does not support. As Root you confirm the conclusion in your integrated result, not in a problem report: satisfied when no model problem remains open; needs_correction when you confirm a genuine finding, your own or an accepted child's, and you dispute every open finding you do not confirm. A confirmed finding ends the Run without realization so that the model can be corrected for a new Run."

const assessGuidance = "A genuine finding here is a model problem: a concrete, material defect of the bound model with its basis. Material means that the model's purpose depends on it: realizations that both follow the model would give different results, acceptances or refusals that users or other areas rely on, or could not be realized or verified at all, and neither derivation from the model nor delegated freedom settles which. A missing decision or an unusable responsibility or contract takes correction model_gap; statements that cannot all hold, or an erroneous statement, take correction model_error. Its obligation names the Definition; its observed text names the source Definitions and statements, the incompatible outcomes or the contradiction, and why neither derivation nor delegated freedom settles it. A suggestion is no genuine finding: missing optional detail, an ordinary implementation choice the model leaves out, an unusual or unfashionable but consistent design, an explicitly delegated choice, wording, style, best-practice claims, preferences and unsupported doubt. Record a suggestion as a note with blocking false, or leave it out; it never blocks, and you never soften a model problem into a note. Dispute a finding of another reviewer only when you independently judge it unfounded, with your reason and the model text as evidence; a real model problem with the wrong class is no dispute: its parent returns it for the class to be corrected."

// assessSpec is the Model assessment Pass. Its correction purposes are the
// classes of a model problem, which no Pass of the run corrects; an omitted
// class takes the last one and stays visible.
func assessSpec() run.PassSpec {
	return run.PassSpec{Purpose: run.Assess, Mandate: assessMandate, Review: true, Assesses: true, Guidance: assessGuidance, Corrections: []run.Purpose{"model_error", "model_gap", "unclassified"}}
}

// extractAssessSpec is the Model assessment that begins an Extract.
func extractAssessSpec() run.PassSpec {
	spec := assessSpec()
	spec.Mandate, spec.Guidance = extractAssessMandate, extractAssessGuidance
	return spec
}

// assess runs a Model assessment Pass; a Pass that only fresh disputes keep
// from being clean is repeated on the unchanged stand within the Pass budget,
// as Verify is. It reports whether the assessment accepted and, when it
// neither accepted nor ended the run, why the run is incomplete.
func assess(r *run.Run, spec run.PassSpec) (bool, string) {
	res := r.Pass(spec)
	for !res.Stopped && res.Verdict == run.NeedsCorrection && r.Reassess() {
		res = r.Pass(spec)
	}
	switch {
	case res.Stopped:
		return false, ""
	case res.Verdict == run.Satisfied:
		return true, ""
	}
	return false, "the model assessment did not reach a clean Pass"
}

// AssessmentAccepted is the reason of an AssessCaptured run whose Model
// assessment accepted the model: it is Incomplete only because nothing is
// realized after the assessment.
const AssessmentAccepted = "the model assessment accepted the model; this entry realizes nothing"

// AssessCaptured runs only the Model assessment of the captured stand, with
// the same Briefs, ownership, gates and budget as Apply in the same closed run
// but without work, required checks or publication: the evaluation entry of
// the assessment method's trial; no Command uses it. Its RunBinding names the
// assessment as its only Pass. The shared work volume and the provider homes
// are removed afterwards.
func AssessCaptured(ctx context.Context, req ApplyRequest) (ApplyResult, error) {
	cfg, binding, err := applyConfig(req, req.now(), entryAssess)
	if err != nil {
		return ApplyResult{}, err
	}
	if err := requireVolumeSubpaths(ctx, req.RequireVolumeSubpaths); err != nil {
		return ApplyResult{Binding: binding}, err
	}
	r, err := run.Open(ctx, cfg)
	if err != nil {
		return ApplyResult{Binding: binding}, err
	}
	accepted, reason := assess(r, assessSpec())
	if accepted {
		reason = AssessmentAccepted
	}
	outcome, runErr := r.Finish(reason)
	ports := req.ports()
	ports.removeHomes(ctx, outcome.Homes)
	ports.closeWork(ctx)
	return ApplyResult{Binding: binding, Outcome: outcome}, runErr
}
