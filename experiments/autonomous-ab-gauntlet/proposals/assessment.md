# Forward planning and completion verification: design input

Status: proposal assessment, not an activated product or benchmark contract. The owner supplied the [supplement](forward-plan-backward-gate.md) on 2026-10-05 as a possible direction for addressing workflow discipline. Its original bytes are retained with SHA-256 `7721efa9a0b55166d66cc08a530aa2fa06685b8ca6abc5d6816b19b808ebc64b`. Imperatives in that attachment are design input; they do not establish that every proposed mechanism is justified or implemented.

The running development cohort still uses product source `adef79d935399f8ac63ad874dbdeab8d15c418a1` and its frozen binaries. Preserve failures, exclusions and deadlines. No proposal is supplied to scored actors. Finish feasible development assessment and classify failures before the existing global fix/no-fix decision. Reserved tasks and outcomes remain closed. Any later product fix needs its own source, tests, candidate identity and unchanged holdout evaluation; the original development results are never rescored.

A separate [source-only review](independent-review-v1.md) reaches the same bounded conclusion without inspecting arena or holdout outcomes: the workflow gap is plausible, but existing command composition should be tested before a persistent product plan format. Its [original bytes](independent-review-v1.raw.json) have SHA-256 `2bc5f09bfd9ff4350713285ad7fb73037e98d8bc01f83ece84d33866df6d7e9b`; the readable Markdown view removes two trailing-space line breaks for the repository whitespace gate and has SHA-256 `46df87a900f756699a48c7a82d7e17c8090db1a693dc0d42656d6b18a197d120`. The original remains unchanged.

## What the proposal addresses

The [Markitect-first protocol](../../../docs/markitect-first.md) already asks an agent to distinguish implementation-only work from engineering-intent changes and establish desired state first. The additional hypothesis is that remembering that workflow is insufficient: an independently enforced completion operation should compose semantic validation, configured implementation evidence, artifact accounting and projection verification.

That is compatible with the [Host/Module boundary](../../../docs/development/modules.md). Host can compose existing independent evidence providers. Core must not learn source-language semantics, task scheduling, Git history or provider details. A new Domain operator or a new Module is not required merely because the operation has a useful name.

| Failure class | What a completion mechanism could establish | Limit |
|---|---|---|
| Implementation introduces a concept absent from the canonical model | A configured specialist/project check can compare its declared implementation inventory with modeled owners | Artifact path accounting alone does not prove UseCase/Handler/Validator ownership |
| A generated provider surface becomes stale | Existing projection checks can compare exact desired and observed bytes | Agreement does not authorize the underlying architecture decision |
| A changed managed file has no accounted owner | Existing artifact coverage can identify the ownership gap | An owned file can still contain an unmodeled or incorrect implementation |
| Required check cannot start | Preserve incomplete evidence and refuse completion | A new plan does not repair PATH, access refusal or tool availability |
| Controller loses a repair opportunity or mislabels a run | Correct the experiment/controller separately and retain the incident | This is not evidence that Markitect needs a product primitive |
| Agent changes intent merely to obtain green checks | An independently retained pre-implementation input binding can make the original plan stale | Self-authored replacement plans and Git ordering alone do not establish owner acceptance |

These are hypotheses about mechanism scope, not a final classification of the current results. In particular, the available public executable failures have not yet been causally separated into product, environment, harness and actor-configuration causes.

## Trust and evidence limits to resolve before implementation

A deterministic digest proves that referenced bytes match. It does not authenticate a reviewer, approve a task classification, or prove when implementation began. A credible desired-state-first contract must identify who retains the planning record before the implementation turn, which accepted revision/configuration authorizes it, how CI receives that original binding, and who may replan. Hashing a mutable candidate-owned plan after the change would not prevent retroactive justification.

Implementation-only work should bind the accepted BASE model without invented YAML edits. Engineering-intent work needs a separately recorded desired-state checkpoint before implementation. Replanning must be visible, with the old record retained. A read-only migration analysis remains policy-failing; it cannot become a passing acceptance result without repair or an explicit valid existing exception.

Impact supplies semantic dependencies and conservative causes. It is neither a source-code inventory nor a complete allowed-file list. A future surface check must keep directly expected changes, conservatively affected resources, explicitly allowed supporting artifacts and unexplained changes separate. An adopter owns the permitted technical scope and required checks; an LLM explanation is not proof that an unexpected change is allowed.

Reuse existing normalized model and fixed-snapshot verification rather than creating a second policy evaluator. Preserve each provider's native findings and unavailable evidence. Host may aggregate a completion result, but should not flatten a missing check, failed check, stale input, policy waiver and owner decision into one unexplained boolean. Local and CI execution must name their exact inputs and tools; a locally passing result is not a CI receipt.

## Smallest evidence-driven next step

1. Finish and retain development evidence, including controller/tooling incidents and excluded trajectories.
2. Classify recurring failures without attributing every failure to agent discipline. Reproduce suspected generic defects outside scored runs.
3. At the global decision, evaluate whether existing checks lack composition, whether a project-owned implementation check is missing, or whether an input-bound execution plan adds demonstrable value. A no-fix decision remains valid.
4. If justified, first compose existing checks for one bounded failing case and an implementation-only passing control. Add persisted plan semantics only where an independently retained checkpoint is actually needed. Do not implement the entire supplement by default.
5. Test false confidence, unavailable tools, stale bindings, forbidden auto-waivers, projection drift and legitimate supporting changes before a new candidate freeze. Use the unchanged reserved holdout under the existing protocol.

Falsification matters: record legitimate changes rejected, manual plan upkeep, model edits made solely to appease the gate, broad Impact interpretation, duplicated architecture checks and missed drift. Do not weaken requirements to obtain green results. [Measurement](../../../docs/measurement.md) remains the owner of benefit-evidence boundaries; no human-attention, productivity or token-saving claim follows from this design note.
