# Model assessment: method and contract for the mandatory gate

**Session C's contract for Session A, 21 October 2026.** These pages own the behavior:
- The [accepted workflow](../../docs/design/canonical-model/manager-apply-workflow.md#mandatory-model-assessment-and-closed-runs) makes Model assessment a mandatory step.
- Its [conformance mandate](../../docs/design/canonical-model/manager-apply-workflow.md#conformance-mandate-and-observation-kinds) sets the materiality test and the observation kinds.
- The [handoff](../../docs/development/claude-direct-runtime-handoff-20261021.md#session-c-model-assessment-before-realization) gives C the method and A the production integration.

This page states what C's isolated harness implements and what the production gate must keep or add. It adds no behavior of its own. The trial's results, not this page, show how well the method finds gaps without false alarms.

## Place in the Run

Bind authority and limits, deterministic Model check, strict File check, then **one** Model assessment, then the selected work composition. The assessment runs once per Run on the bound model. It does not assess itself, and it does not repeat before every Turn or before an artifact-correction Pass against unchanged authority. A changed model is a new Run with a new assessment.

## The Pass

- **Purpose `assess`, one review Pass of the common recursive process.** Every Manager of the required tree is a fresh read-only reviewer of its own obligations. Parents release children on their accepted basis, then accept, return or dispute child results and integrate interfaces and cross-area consistency. Root's integrated result seals the Pass with one of the two common verdicts. There is no second scheduler, state or verdict.
- **Materiality.** Is the model usable, materially consistent and decisive enough for its realization? Not: is it as detailed as possible, or does it follow a preferred architecture? A decision is material when the model's purpose depends on it. Two realizations that both follow the model would differ in results, acceptances or refusals that users or other areas rely on, or could not be realized or verified at all, and neither derivation nor delegated freedom settles which. The following are no defects: missing optional detail, ordinary implementation choices left out, an unusual but consistently stated design, explicitly delegated choices.
- **Treatment.** The mandate and the guidance of [assessment-pass.json](assessment-pass.json), bound by its hash. They ask about the outcomes others rely on, consistency, usable responsibility and contracts, and the permitted freedom. Evidence is model sources, and there is no quota.
- **No required check, probe, write or publication.** A check judges artifacts, not the model. Artifacts in the stand are not assessed and never settle what the model leaves open.
- **Closed Run.** No owner channel, no earlier answers and no owner decisions. Questions travel only along the Manager tree and may make a Manager Waiting. A material decision the model lacks becomes a model problem, never a question to the user, a default or a chosen value.
- **Disputes.** Root confirms the findings it keeps and disputes every open finding it does not confirm. While only fresh disputes keep the Pass from being clean, a new complete assessment Pass of the unchanged stand decides them, within the Pass budget, as Verify does in Apply.

## Observation kinds

The assessment uses the workflow's common kinds and existing dispositions, with no new record or backlog:

| Kind | In the assessment | Effect |
| --- | --- | --- |
| Confirmed deviation | Does not apply: no artifact is assessed. | — |
| Suspected deviation | A concrete suspected model problem. The reviewer investigates it inside the Pass by lookup and questions along the tree, and then classifies it, resolves it or keeps working. | The reviewer cannot report its result as satisfied, or park the suspicion as a note, while it is unresolved; at the limit the Run is `Incomplete`. |
| Model problem | A genuine finding with its basis: the Definition as obligation; the source statements, the incompatible outcomes or the contradiction, and why derivation or delegated freedom cannot settle it as observed text; the class in `correction`. | Root's confirmation seals `NeedsCorrection` and ends the Run. |
| Suggestion | A non-blocking note, or nothing at all. | Never blocks. |

Disputes, returns and disproofs are the dispositions. A resolved or disproved finding is not raised again on the same statements without new evidence. When A records the kinds as explicit observation fields, the assessment's records map to them one to one; until then they are the existing finding, note and coverage fields.

## Payload and verdict

A finding is **confirmed** when it is open, genuine, raised by the team (not `markitect`) and not disputed in the sealed Pass.

| Conclusion | Common envelope | Run |
| --- | --- | --- |
| `accepted` | Sealed `Satisfied`: no open genuine finding of the team. | Releases the work composition. |
| `model_gap` | Sealed `NeedsCorrection` with confirmed findings, at least one classed `model_gap` and none `model_error`. | `Failed`, no publication, no in-Run repair. |
| `model_error` | As above, with at least one confirmed `model_error`. | `Failed`, no publication, no in-Run repair. |
| (unclassed) | Confirmed findings whose class was omitted. | `Failed` like the others; a protocol defect of the report, recorded as such. |
| (incomplete) | The Pass did not seal: deadline, turn or Pass limits, an uncertain turn, evidence or port failures; or a sealed negative Pass without a confirmed finding. A failed deterministic gate stays its own failure. | `Incomplete`, or its own `Failed` classification; never a model conclusion. |

The class rides in the existing `correction` field, because an assessment has no in-Run correction purpose. The corrections are `[model_error, model_gap, unclassified]`, and the tool fills an omitted value with the last one. An omitted class therefore stays visible instead of passing for a gap. Production may give the class its own payload field.

Root's model-error decision (`problem`, category `model`, `abort`) during the assessment ends the Run `Failed` as before. The mandate asks Root to conclude in its integrated result instead, so that the confirmed findings stay in the sealed Pass. The harness records an abort as a second route, classed `model_error` unless its free text names `model_gap`.

## What production must add

The harness entry `apply.AssessCaptured` (with `TestModelAssessmentHarness` in `internal/bootstrap`) reuses `applyConfig`, `run.Open`, `Pass`, `Reassess` and `Finish`. It changes no Apply behavior. For the mandatory gate, A owns these changes:

1. **Terminal `Failed` for a sealed negative assessment.** `run.Finish(reason)` turns a caller's reason into `Incomplete`. The harness therefore ends a Pass sealed `NeedsCorrection` with open genuine findings `Incomplete`, with the reason `the model assessment confirmed a material model problem`. A sealed negative Pass without open genuine findings ends with *"the model assessment did not reach a clean Pass"*. The harness classifies from the sealed Pass. The gate needs a definite-rejection end for the first case.
2. **Release only on `Satisfied`.** Run the strategy's work Passes only after an accepted assessment. Production assessment uses the same bound capture, and nothing resumes a failed Run.
3. **The closed-Run refusal text.** With no owner channel, `send` and Root's `escalate` to the user are refused with *"Decide within the model and the bound owner decisions; a model that is erroneous or contradictory is reported with report kind problem, category model, abort true"*. Under the closed-Run policy this invites choosing a value for a missing decision. In an assessment it should point to a `model_gap` finding, and in work Passes to Root's confirmed gap stop. Remove the mention of owner decisions.
4. **Assessment framing and Root's duty.** Review Passes show no `RootGuidance` and frame every reviewer as assessing "an immutable candidate", so the assessment's Root duty lives in the mandate. Briefs still list the required checks, which do not run in the assessment. The work mandates of both strategies still tell Managers to ask the user about missing public choices; under the closed-Run policy they belong to A's migration.
5. **Evidence binding.** The harness already binds through A's `RunBinding`: the entry is `model-assessment`, with no work Pass and no check, the classes as corrections and the assessment Pass's digest as composition. Production should keep the assessment digest beside the strategy's and record the conclusion with the Run. It should also record:
   - coverage: every own obligation `done` with a model-source evidence item;
   - dissent: returns, disputes and disproofs;
   - actual context delivery: the harness's scorer checks each reviewer's first sent prompt against the frozen Definitions' identities and bodies. The workflow's coverage record should make this a product observation.
6. **Observation kinds.** Map the assessment's model problems, suspicions and suggestions onto the common observation fields when A introduces them. Keep the rule that code checks declared effects and provenance, not the truth of an Agent's classification.
7. **Extract.** Assess the available mandate, model and constraints before Extract work, and the resulting candidate when it exists, with the same Pass and conclusions. The harness does not cover Extract.

C's source tests in `internal/apply/assessment_test.go` cover the harness entry:
- read-only fresh reviewers without checks or publication;
- the binding with its own composition digest;
- Root's model-error end;
- a closed Run that refuses a user question and keeps owner decisions out of every prompt;
- a confirmed finding that seals `NeedsCorrection` with its class;
- reassessment of a disputed finding;
- refusal of a writing or non-review Pass.

The handoff lists A's own production controls.

## Evaluation

[README.md](README.md) holds the method, the controls and the measures of the trial. Its report belongs in `docs/validation` after the owner's start signal.

- The complete and freedom controls must be accepted even when they omit optional detail or state an unconventional valid choice.
- The gap and contradiction controls must still be rejected.
- An accepted assessment is no completeness proof: a later Root-confirmed problem still stops the Run, and independent artifact Verify still follows realization.
