# Model assessment: method and contract for the mandatory gate

**Session C's contract for Session A, 21 October 2026.** These pages own the behavior:
- The [accepted workflow](../../docs/design/canonical-model/manager-apply-workflow.md#mandatory-model-assessment-and-closed-runs) makes Model assessment a mandatory step.
- Its [conformance mandate](../../docs/design/canonical-model/manager-apply-workflow.md#conformance-mandate-and-observation-kinds) sets the materiality test and the observation kinds.
- The [handoff](../../docs/development/claude-direct-runtime-handoff-20261021.md#session-c-model-assessment-before-realization) gives C the method and A the production integration.

This page states the method exercised by C's isolated harness through A's production assessment. It adds no behavior of its own. The trial's results, not this page, show how well the method finds gaps without false alarms.

## Place in the Run

Bind authority and limits, deterministic Model check, strict File check, then **one** Model assessment, then the selected work composition. The assessment runs once per Run on the bound model. It does not assess itself, and it does not repeat before every Turn or before an artifact-correction Pass against unchanged authority. A changed model is a new Run with a new assessment.

## The Pass

- **Purpose `assess`, one review Pass of the common recursive process.** Every Manager of the required tree is a fresh read-only reviewer of its own obligations. Parents release children on their accepted basis, then accept, return or dispute child results and integrate interfaces and cross-area consistency. Root's integrated result seals the Pass with one of the two common verdicts. There is no second scheduler, state or verdict.
- **Materiality.** Is the model usable, materially consistent and decisive enough for its realization? Not: is it as detailed as possible, or does it follow a preferred architecture? A decision is material when the model's purpose depends on it. Two realizations that both follow the model would differ in results, acceptances or refusals that users or other areas rely on, or could not be realized or verified at all, and neither derivation nor delegated freedom settles which. The following are no defects: missing optional detail, ordinary implementation choices left out, an unusual but consistently stated design, explicitly delegated choices.
- **Treatment.** The production `assessSpec()` mandate and guidance, with [assessment-pass.json](assessment-pass.json) as a hashed reference checked against actual reviewer prompts. The reference never supplies an execution specification. They ask about the outcomes others rely on, consistency, usable responsibility and contracts, and the permitted freedom. Evidence is model sources, and there is no quota.
- **No required check, probe, write or publication.** A check judges artifacts, not the model. Artifacts in the stand are not assessed and never settle what the model leaves open.
- **Closed Run.** No owner channel, no earlier answers and no owner decisions. Questions travel only along the Manager tree and may make a Manager Waiting. A material decision the model lacks becomes a model problem, never a question to the user, a default or a chosen value.
- **Disputes.** A dispute is only for an unfounded finding. A real problem with the wrong class is returned to its raiser for the class, so that a disproof can never remove a real gap. Root confirms the findings it keeps and disputes every open finding it does not confirm. While only fresh disputes keep the Pass from being clean, a new complete assessment Pass of the unchanged stand decides them, within the Pass budget, as Verify does in Apply.

## Observation kinds

The assessment uses the workflow's common kinds and existing dispositions, with no new record or backlog:

| Kind | In the assessment | Effect |
| --- | --- | --- |
| Confirmed deviation | Does not apply: no artifact is assessed. | None. |
| Suspected deviation | A concrete suspected model problem. The reviewer investigates it inside the Pass by lookup and questions along the tree, and then classifies it, resolves it or keeps working. | The reviewer cannot report its result as satisfied, or park the suspicion as a note, while it is unresolved; at the limit the Run is `Incomplete`. |
| Model problem | A genuine finding with its basis: the Definition as obligation; the source statements, the incompatible outcomes or the contradiction, and why derivation or delegated freedom cannot settle it as observed text; the class in `correction`. | Root's confirmation seals `NeedsCorrection` and ends the Run. |
| Suggestion | A non-blocking note, or nothing at all. | Never blocks. |

Disputes, returns and disproofs are the dispositions. A resolved or disproved finding is not raised again on the same statements without new evidence. When A records the kinds as explicit observation fields, the assessment's records map to them one to one; until then they are the existing finding, note and coverage fields.

## Payload and verdict

A finding is **confirmed** when it is open, genuine, raised by the team (not `markitect`) and not disputed in the sealed Pass.

| Conclusion | Common envelope | Run |
| --- | --- | --- |
| `accepted` | Sealed `Satisfied`: no open genuine finding of the team. | Releases production work. Assessment-only entry ends `Incomplete` with `AssessmentAccepted`, realizing and verifying nothing. |
| `model_gap` | Sealed `NeedsCorrection` with confirmed findings, at least one classed `model_gap` and none `model_error`. | `Failed`, no publication, no in-Run repair. |
| `model_error` | As above, with at least one confirmed `model_error`. | `Failed`, no publication, no in-Run repair. |
| (unclassed) | Confirmed findings whose class was omitted. | `Failed` like the others; a protocol defect of the report, recorded as such. |
| (incomplete) | The Pass did not seal: deadline, turn or Pass limits, an uncertain turn, evidence or port failures; or a sealed negative Pass without a confirmed finding. A failed deterministic gate stays its own failure. | `Incomplete`, or its own `Failed` classification; never a model conclusion. |

The class rides in the existing `correction` field, because an assessment has no in-Run correction purpose. The corrections are `[model_error, model_gap, unclassified]`, and the tool fills an omitted value with the last one. An omitted class therefore stays visible instead of passing for a gap. Production may give the class its own payload field.

Root's model-error decision (`problem`, category `model`, `abort`) during the assessment ends the Run `Failed` as before. The mandate asks Root to conclude in its integrated result instead, so that the confirmed findings stay in the sealed Pass. The harness records an abort as a second route, classed `model_error` unless its free text names `model_gap`.

## Production integration

A integrated the production hook in [#566](https://github.com/Glacius-Labs/Markitect/pull/566), Main `54fc867032aa8e38a368ae1c3dd87e77171236dc`. C uses `apply.AssessCaptured(ctx, req)`, which runs the built-in `assessSpec()` through the same recursive Pass as mandatory Apply assessment. C's provisional second implementation and the caller-supplied Pass specification are removed. The harness has no Owner/OwnerDecisions or environment-supplied treatment alternative.

The production mandate and guidance are exactly the corrected C reference from `7db130d7`, SHA-256 `ae7d3a4b2d2e34707d88df5f28565451dfcf1ed825470e7de6242a4554521a02`. C's source tests compare that reference with the actual reviewer prompts; the omitted-class test checks the production default remains visible. The reference is never passed to execution.

A Root-confirmed material problem seals `NeedsCorrection` and ends `Failed` with `run.ModelAssessmentReason`, without publication or model repair. Acceptance seals `Satisfied`; in normal Apply it releases the selected strategy's work. In the assessment-only entry it ends `Incomplete` with `apply.AssessmentAccepted`, since no artifact was realized or independently verified. Technical errors, unsealed results and unresolved evidence never become model conclusions. Root's older direct model-problem abort remains a separately recorded route, classified from its report text.

The binding identifies entry `model-assessment`, `Work: [assess]`, no checks, and the production composition digest. Its composition digest includes the assessment and strategy declarations; it is distinct from the reference file's byte hash. Context binding records the Brief and full/reference identities; C's transcript comparison records body submission, not provider understanding or semantic coverage.

C's `internal/apply/assessment_test.go` covers the built-in method reference, read-only fresh reviewers, absence of checks/publication, Root's model-error stop, refusal of external questions, confirmed finding classes, the omitted class and fresh reassessment of a disputed finding. `internal/bootstrap/model_assessment_evaluation_test.go` supplies the grant-gated adapter wiring. These source tests do not establish provider behavior or a completed wiring smoke. Extract's production migration and evidence remain A's responsibility; this model-only harness does not exercise Extract.

## Evaluation

[README.md](README.md) holds the method, the controls and the measures of the trial. Its report belongs in `docs/validation` after the owner's start signal.

- The complete and freedom controls must be accepted even when they omit optional detail or state an unconventional valid choice.
- The gap and contradiction controls must still be rejected.
- An accepted assessment is no completeness proof: a later Root-confirmed problem still stops the Run, and independent artifact Verify still follows realization.

The primary-goal clarification requires reviewers to derive material clauses, exceptions and cross-area conditions independently from the bound model and its explicit purpose. Complete Definition IDs and an author's coverage list do not establish that work. Parent acceptance must challenge actual contract consistency and unsupported acceptance as well as reported problems. An alleged implementation freedom needs a model-based reason; it cannot erase an unresolved public decision. This interpretation does not change the frozen treatment or claim that its runtime enforcement has been demonstrated.

A later bounded adequacy/authoring exercise would start from supplied project purpose and requirements and assess whether the authored concepts, relations, interfaces and cross-area rules express them usefully. It would record omissions, invented constraints and owner effort separately from internal consistency. It needs its own scope and controls; it adds no fifth control, provider attempt, framework or longitudinal study to this assignment. Without supplied purpose, an entirely omitted need remains outside what this model-only assessment can discover.
