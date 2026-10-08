# Documentation projection workstream

## Context

Start from the full verified preparation integration SHA supplied in the assignment, as defined by [Baseline](../development/baseline.md). PR 59's `e9550f5` is the source audit base only; it omits these preparation contracts and is not the dispatch baseline.

At the PR 59 integrated baseline (`e9550f5c91430c6a65cbbd4ffcdbb49b48272537`), Markitect has a built-in `markdown` target that writes resource and Domain views. It is not an external command adapter and does not consume the command `SemanticModel` DTO. Shared assumptions are owned by [the shared adapter contracts](../development/shared-contracts.md).

## Objective

Clarify or improve the built-in documentation projection while preserving canonical ownership, deterministic output, and explicit target selection.

## Scope

This workstream covers Markdown output selected by `Project.spec.targets: [markdown]`, generated view/navigation paths and their renderer tests. Any proposed connection to an external documentation service must first define desired state, mapping ownership, target identity and evidence; that would use the external command boundary and remain a separate integration.

## Current implementation

`internal/render` consumes the validated graph and captured files to generate opt-in Markdown views under `docs/markitect/`, including Domain views and navigation. Outputs are derived from canonical resources/Domain definitions. The local `markitect-render` reconciliation path observes, plans, applies and verifies generated file bytes. Generated Markdown is not canonical input to provider adapters.

## Owned subsystem

Documentation-specific projection code and focused renderer tests may be owned here when isolated in their own files. `internal/render/render.go`, shared output path/collision logic, Markdown navigation planning and reconciliation are shared with other renderer targets and require coordinator review.

## Allowed changes

- Clarify the existing generated Markdown contract and evidence limits.
- Add documentation-specific helper files or tests with separate ownership.
- Propose deterministic view content or path behavior with explicit output ownership and collision handling.
- Explore external Docs publication only after desired-state and write-target boundaries are specified; start read-only and target-scoped.

## Forbidden changes

- Do not infer typed dependencies, ownership, or semantic meaning from prose links.
- Do not make generated Markdown canonical or create adapter chains that treat another adapter's output as truth.
- Do not modify shared renderer dispatch/output ownership without coordinator review.
- Do not add Core fields, implicit targets, background publishing, or external writes in `check`, `model`, `observe`, `plan`, or `verify`.

## Dependencies

Depends on normalized graph/render inputs, explicit Project target selection, and local output ownership rules. Shared renderer changes depend on coordinated Codex/Claude review. See [shared adapter contracts](../development/shared-contracts.md).

## Design questions

- Is this a generated local Markdown projection or an external Docs system integration?
- What canonical resources and Domain inputs own each generated file?
- Are links navigation only, or are they typed graph relationships? The renderer must not infer the latter.
- If an external target is involved, which exact object/path is owned, how are competing writers prevented, and what can observation actually establish?
- Is desired state based directly on canonical model data rather than another generated representation?

## Required tests

For local output changes, test deterministic bytes, explicit target gating, ownership/path collisions, stale output reporting, and links remaining navigation-only. For any external command integration, add independent protocol tests for observe/plan/verify, declared captured inputs and observation-bound planning; test writes only if separately authorized and designed.

## Required evidence

Record source revision, target selection, canonical owner and source digests for local outputs, generated path/hash inventory, drift/stale/collision findings, and test commands. External integrations additionally record exact target identity, adapter executable hash, scope, and observation limits.

## Exit criteria

Every output has an explicit canonical owner and deterministic path; collisions or stale files are surfaced; generated docs remain derived; tests establish only the documented structural behavior.

## Completion questions

- Can each changed file be traced to one canonical resource, Domain, or Project-owned navigation rule?
- Did any output become an input to another adapter's desired-state computation?
- Were shared renderer and output-planning edits reviewed as shared work?
