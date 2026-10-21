# Model assessment: method and contract for the mandatory gate

**Session C's contract for Session A, 21 October 2026.** The [accepted workflow](../../docs/design/canonical-model/manager-apply-workflow.md#mandatory-model-assessment-and-closed-runs) makes Model assessment a mandatory step and owns its behavior; the [handoff](../../docs/development/claude-direct-runtime-handoff-20261021.md#session-c-model-assessment-before-realization) gives C the method and A the production integration. This page states what C's isolated harness implements and what the production gate must keep or add. It adds no behavior of its own, and the trial's results, not this page, show how well the method finds gaps without false alarms.

## Place in the Run

Bind authority and limits, deterministic Model check, strict File check, then **one** Model assessment, then the selected work composition. The assessment runs once per Run on the bound model. It does not assess itself, and it does not repeat before every Turn or before an artifact-correction Pass against unchanged authority. A changed model is a new Run with a new assessment.

## The Pass

- **Purpose `assess`, one review Pass of the common recursive process.** Every Manager of the required tree is a fresh read-only reviewer of its own obligations. Parents release children on their accepted basis, then accept, return or dispute child results and integrate interfaces and cross-area consistency. Root's integrated result seals the Pass with one of the two common verdicts. There is no second scheduler, state or verdict.
- **Treatment.** The mandate and the guidance of [assessment-pass.json](assessment-pass.json), bound by its hash. They ask for public decisions, consistency, usable responsibility and contracts, and the permitted freedom, with model sources as evidence and no quota.
- **No required check, probe, write or publication.** A check judges artifacts, not the model. Artifacts in the stand are not assessed and never settle what the model leaves open.
- **Closed Run.** No owner channel, no earlier answers and no owner decisions. Questions travel only along the Manager tree and may make a Manager Waiting. A material decision the model lacks becomes a finding, never a question to the user, a default or a chosen value.
- **Disputes.** While only fresh disputes keep the Pass from being clean, a new complete assessment Pass of the unchanged stand decides them, within the Pass budget, as Verify does in Apply.

## Payload and verdict

| Conclusion | Common envelope | Run |
| --- | --- | --- |
| `accepted` | Sealed `Satisfied`: no open genuine finding of the team. | Releases the work composition. |
| `model_gap` | Sealed `NeedsCorrection`: Root's integrated result confirms open genuine findings (its own, or accepted child findings that were not disproved); none is classed `model_error`. | `Failed`, no publication, no in-Run repair. |
| `model_error` | As above, with at least one finding classed `model_error`. | `Failed`, no publication, no in-Run repair. |
| (technical) | The Pass did not seal: deadline, turn or Pass limits, an uncertain turn, evidence or port failures. A failed deterministic gate stays its own failure. | `Incomplete`, or its own `Failed` classification; never a model conclusion. |

Root's model-error decision (`problem`, category `model`, `abort`) during the assessment ends the Run `Failed` as before. The mandate asks Root to conclude in its integrated result instead, so that the confirmed findings stay in the sealed Pass. The harness treats an abort as a second route and records it as such. Orchestrator findings (`raisedBy: markitect`) never count as model findings. A finding's class rides in the existing `correction` field, with the corrections `[model_error, model_gap]`, because an assessment has no in-Run correction purpose. Production may give the class its own payload field.

A confirmed finding names the Definition as its obligation. Its observed text names the source Definitions and statements, the incompatible outcomes or the contradiction, and why neither derivation nor delegated freedom settles it. Remarks about permitted freedom are non-blocking notes.

## What production must add

The harness entry `apply.AssessCaptured` (with `TestModelAssessmentHarness` in `internal/bootstrap`) reuses `applyConfig`, `run.Open`, `Pass`, `Reassess` and `Finish`. It changes no Apply behavior. For the mandatory gate, A owns these changes:

1. **Terminal `Failed` for a sealed negative assessment.** `run.Finish(reason)` turns a caller's reason into `Incomplete`, so the harness ends a confirmed problem `Incomplete` with the reason `the model assessment confirmed a material model problem` and classifies from the sealed Pass. The gate needs a definite-rejection end for it.
2. **Release only on `Satisfied`.** Run the strategy's work Passes only after an accepted assessment. Production assessment uses the same bound capture, and nothing resumes a failed Run.
3. **The closed-Run refusal text.** With no owner channel, `send` to the user is refused with *"Decide within the model and the bound owner decisions; a model that is erroneous or contradictory is reported with report kind problem, category model, abort true"*. Under the new policy this invites choosing a value for a missing decision. In an assessment it should point to a `model_gap` finding, and in work Passes to Root's confirmed gap stop. Remove the mention of owner decisions.
4. **Assessment framing and Root's duty.** Review Passes show no `RootGuidance` and frame every reviewer as assessing "an immutable candidate"; the assessment's Root duty therefore lives in the mandate. Briefs still list the required checks, which do not run in the assessment.
5. **Evidence binding.** The harness already binds through A's `RunBinding`: the entry is `model-assessment`, with no work Pass and no check, the classes as corrections and the assessment Pass's digest as composition. Production should keep the assessment digest beside the strategy's and record the conclusion with the Run. It should also record coverage (every own obligation `done` with a model-source evidence item) and dissent (returns, disputes, disproofs). Record actual context delivery: the harness's scorer checks each reviewer's first sent prompt against the frozen Definitions' identities and bodies, which the workflow's coverage record should make a product observation.
6. **Extract.** Assess the available mandate, model and constraints before Extract work and the resulting candidate when it exists, with the same Pass and conclusions. The harness does not cover Extract.

C's source tests in `internal/apply/assessment_test.go` cover the harness entry: read-only fresh reviewers without checks or publication, the binding with its own composition digest, Root's model-error end, a closed Run that refuses a user question and keeps owner decisions out of every prompt, a confirmed finding sealing `NeedsCorrection` with its class, reassessment of a disputed finding, and refusal of a writing or non-review Pass. The handoff lists A's own production controls.

## Evaluation

[README.md](README.md) holds the method, the controls and the measures of the trial. Its report belongs in `docs/validation` after the owner's start signal. An accepted assessment is no completeness proof: a later Root-confirmed problem still stops the Run, and independent artifact Verify still follows realization.
