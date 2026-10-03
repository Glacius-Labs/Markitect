# Adapter contract and independence

## Context

Start from the full verified preparation integration SHA supplied in the assignment, as defined by [Baseline](../development/baseline.md). PR 59's `e9550f5` is the source audit base only; it omits these preparation contracts and is not the dispatch baseline.

Read [shared contracts](../development/shared-contracts.md) and the [audit](../development/parallelizability.md). External commands have a versioned normalized-model seam. Native rendering uses Graph/files; cross-command target exclusivity is not enforced centrally.

## Objective

Freeze the current seam for a wave and prove two unrelated consumers work independently. Resolve only concrete shared-contract questions required by assigned consumers.

## Scope

Contract fixtures, evidence limits, ownership assessment and focused design. No broad plugin runtime or new adapters in contract preparation. No blocking Core defect has been established.

## Current implementation

`internal/app/model.go`, `adapter_command.go`, `reconcile.go`, `cmd/markitect/cli_reconcile.go`, `internal/render`. Actual protocol labels are `v1alpha1`; command Observe/Plan/Verify are mandatory, Apply optional and explicit. Native `markitect-render` uses a separate path.

## Owned subsystem

Coordinator owns model/DTOs, lifecycle/digests and shared target semantics. This workstream owns assigned contract tests/design; shared edits need coordinator approval and cannot be self-approved.

## Allowed changes

`docs/development/shared-contracts.md`, `parallelizability.md`, isolated new protocol fixtures explicitly assigned. A code repair requires a reviewed defect/design and separately bounded ownership.

## Forbidden changes

Other adapters, provider Core fields, new operators, implicit capabilities, generated-output chains, universal sandbox/convergence claims, release artifacts.

## Dependencies

Read-only consumers may use today's protocol now. Shared/remote writes wait for target identity/alias conflicts and observation preconditions. Native extensions coordinate shared entrypoints.

## Design questions

Can independent adapters be present/absent without changing Core meaning? Which executable/config inputs bind plans? Which targets overlap? What does observation prove and which remote change stales Apply? Is the smallest remedy provider-owned, contractual or generic?

## Required tests

Two independent executables from one model fixture; absent adapter not invoked; exact inputs/argv; bad response version/model/target; changed executable/config/observation and altered-plan rejection; read-only stages. Test explicit Apply and convergence only for a separately assigned write-capable target; read-only adapters require no mutation capability. New exclusivity semantics require collision/alias tests before enabling writes.

## Required evidence

Exact SHA, protocol fixture bytes, target ownership, complete/incomplete observations, command identity, stale-plan cases, capability and evidence limits. Declared versions are not release attestations.

## Exit criteria

Two bounded consumers test independently without another adapter or provider Core changes. Unresolved native/write limits remain explicit gates; all shared changes receive affected-consumer review.

## Completion questions

Is the seam sufficient? Which central edits remain? Is ownership enforced or reviewed? Did provider meaning enter Core? Which consumers and mutation boundaries were checked?
