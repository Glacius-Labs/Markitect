# Risk triage and regression-evidence workstream

## Context

Start from the full verified preparation integration SHA supplied in the assignment, as defined by [Baseline](../development/baseline.md). PR 59's `e9550f5` is the source audit base only; it omits these preparation contracts and is not the dispatch baseline.

This is a future-work specification based on the preparation baseline after PR #59 (`e9550f5c91430c6a65cbbd4ffcdbb49b48272537`). The latest published product remains v0.12.0; source behavior added after that release must be identified as unreleased. Start with the evidence-based [risk register](../development/risk-register.md), the [parallel work guide](../development/parallel-work.md), and the [development coordination guide](../development/README.md). The [engineering constitution](../engineering-constitution.md) remains the canonical product contract.

The register identifies risks, not a feature backlog. A theoretical risk does not justify a product feature. This workstream is limited to coordinator-assigned regression tests and updates to the risk dossier.

## Objective

For each explicitly assigned risk, establish whether current source violates an adopted contract or whether the concern remains a design question or evidence gap. Preserve a minimal, deterministic reproduction as a new test only after coordinator assignment, and update the risk dossier with the exact evidence and remaining limits. Do not turn investigation into an implementation proposal unless the coordinator requests one separately.

## Scope

- Read the exact assigned source baseline and relevant existing tests before proposing a regression case.
- Investigate only the risk IDs and test package/path named in the assignment.
- When authorized, add focused new tests in the assigned test area and update `docs/development/risk-register.md` with classification, reproduction, evidence, and limits.
- Keep test fixtures local, deterministic, and independent of adopter repositories, credentials, live services, and mutable external state.
- Treat published behavior, current source behavior, unreleased behavior, human acceptance, and adopter observations as distinct evidence.

## Current implementation

The current source includes deterministic typed relations, fixed-snapshot context and impact, conservative unknown-input behavior, digest-bound policy exceptions, and explicit output/reconciliation boundaries. Focused evidence is linked per item in the [risk register](../development/risk-register.md). PR #59's three-task adopter comparison concluded inconclusively: it found useful policy-migration explanation but did not establish that added model upkeep pays for itself. Its bounded tests, source/build checks, missing context, broad Impact, and lack of human/runtime acceptance are recorded in [the comparison report](../validation/agents-md-vs-markitect.md).

The [roadmap](../implementation-plan.md) records current source versus release status. Read-only Context/Impact analysis of ordinary policy failures is unreleased source behavior; v0.12.0 remains the published baseline. Neither a green test nor a risk classification proves adopting-project policy acceptance or general product benefit.

## Owned subsystem

Ownership is limited to:

- New, coordinator-assigned regression tests in the specifically named existing package or an explicitly assigned isolated test fixture.
- The risk dossier at `docs/development/risk-register.md`, only for the assigned risk items.

The coordinator owns risk prioritization, product semantics, cross-package contracts, shared state, integration, and release decisions. No source implementation ownership is implied.

## Allowed changes

- Read current code, existing tests, canonical docs, and the exact assigned evidence needed to isolate a concern.
- Produce a minimal reproduction and describe whether it contradicts an adopted contract.
- After explicit coordinator assignment, add a test that fails on the reproduced defect or guards the explicitly named invariant; place it only in the assigned test area.
- Update the assigned risk-dossier entry with source/test links, exact reproduction and verification commands, classification, and known evidence limits.
- Recommend that a reproduced defect be assigned to a future implementation workstream. The recommendation itself does not authorize implementation.

## Forbidden changes

- Do not treat a theoretical risk, awkward fixture, missing capability, or inconclusive adopter observation as a feature request.
- Do not modify unassigned Core/application code, normalized model or shared state, CLI dispatch, schemas, generated files, shared helpers, adapters, or another workstream's files.
- Do not change the engineering constitution, architecture, canonical roadmap, release metadata, version, package pins, or publication state.
- Do not change adopter/consumer repositories, project policy, credentials, or adopter evidence.
- Do not rewrite, regenerate, delete, or reinterpret historical reports, frozen inputs, experiment manifests, PR evidence, or published-release evidence.
- Do not add broad or speculative tests outside the assigned risk. Do not weaken existing assertions to make a candidate pass.
- Do not implement a confirmed defect in this workstream. A confirmed defect requires a minimal reproduction **and a separate coordinator assignment** to the owning implementation workstream.

## Dependencies

- A coordinator assignment naming the risk ID(s), full source baseline SHA, owned new test path/package, and the exact risk-dossier entry that may change.
- The [risk register](../development/risk-register.md), [parallel work guide](../development/parallel-work.md), and applicable canonical contract and ownership map.
- A reproducible local fixture and existing expected contract. External adopter evidence requires explicit scope and access authorization from its owner; it is not presumed by this specification.
- Any request to change shared semantics, schema, protocol, CLI, release, or adopter behavior is a hard handoff to the coordinator and a separately scoped workstream.

## Design questions

- Which exact adopted statement is allegedly violated, and where is its canonical owner?
- Can the behavior be reproduced against the named source SHA with a small local fixture and no external state?
- Does an existing test already cover the invariant, and if so, what concrete case remains uncovered?
- Is the outcome a defect, a missing test, a documentation clarification, a design proposal, or a need for real-world evidence?
- What is the smallest new test that preserves intended behavior without requiring a product code change?
- Which test commands, platform assumptions, and current-versus-release boundaries must accompany the evidence?
- If the reproduction confirms a defect, which separate implementation owner does the coordinator assign?

## Required tests

No tests are run or added merely because this specification exists. After a specific test assignment:

- Add a deterministic regression test for the minimal reproduction in the assigned test area.
- Assert the required contract outcome and the relevant boundary case; retain existing tests and assertions unchanged.
- Run the focused package test and any test command named by the coordinator. Report exact commands and results.
- After integration, the coordinator runs required repository and platform gates against the exact candidate head; earlier results do not transfer to a changed SHA.
- Do not represent a passing unit test as adopter acceptance, source-language understanding, runtime proof, or release evidence unless those were directly tested.

## Required evidence

Record the full baseline SHA, risk ID, canonical contract link, affected source/test paths, minimal input and reproduction steps, expected and actual behavior, exact test command and result, and the resulting candidate test SHA. For any update to the risk dossier, state the classification and distinguish implementation evidence from tests, adopter observations, human decisions, and inference. Preserve applicable historical evidence unchanged. State explicitly when current source and published v0.12.0 differ.

## Exit criteria

- The coordinator assigned the risk item, exact baseline, and test/dossier ownership before changes began.
- The concern is classified from current evidence; a theoretical risk remains a design/evidence item without an implementation task.
- Any claimed defect has a minimal local reproduction and a separate coordinator assignment before product code is changed.
- Only assigned new tests and the assigned risk-dossier entry changed; no shared contract, Core, schema, adapter, release, adopter, or historical files changed.
- Focused test evidence and limitations are recorded against the exact candidate SHA; integration gates remain coordinator-owned.

## Completion questions

- What exact adopted contract and risk ID were assigned, and which source SHA was assessed?
- Is there a minimal reproduction, or is the risk still theoretical or evidence-limited?
- Which new test captures the behavior, and what exact command and result verify it?
- Does the updated dossier cite source and tests while separating current implementation, published release, adopter evidence, human acceptance, and inference?
- Were changes restricted to assigned new tests and the risk dossier?
- If a defect is confirmed, is there a separate coordinator assignment for its implementation owner?
