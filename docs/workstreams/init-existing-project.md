# Existing-project preparation workstream

This is a bounded future-work specification, not authorization to implement a new Init mode. Read it with the [shared contracts](../development/shared-contracts.md) and [adoption boundaries](../development/adoption-boundaries.md). The coordinator owns shared selection and workspace-handoff design; parallel investigation may proceed, but durable implementation must wait for those contracts and an explicit implementation decision.

## Context

Start from the full verified preparation integration SHA supplied in the assignment, as defined by [Baseline](../development/baseline.md). PR 59's `e9550f5` is the source audit base only; it omits these preparation contracts and is not the dispatch baseline.

Published v0.12.0 `init` starts a new Project only when `markitect.yaml` is absent and the target Area does not exist. It previews and can create only `markitect.yaml` plus one Area README. Its refusal to replace or adopt an existing Project is a safety invariant for current behavior. A future existing-project preparation workflow may help a human owner review a control-plane or discovery workspace, but it must not silently extend the current minimal init contract.

The requested outcome includes making detected roots distinguishable from selected and excluded roots and requiring owner scope review before evidence analysis. The shape and persistence of that selection, snapshot identity, and handoff are unresolved shared design, not assumptions for this workstream to freeze.

## Objective

Define and, only after explicit authorization, implement a reviewable preparation operation for an existing repository. The operation should expose candidate roots and scope information for owner review, then prepare an isolated workspace only from an approved selection. Preserve the existing new-Project Init behavior and explicit refusal boundary unless the coordinator approves a separate, exact change.

## Scope

- This workstream owns product behavior for safe preparation and workspace creation in Markitect's Init application path.
- It may investigate repository roots, filesystem safety, Git snapshot acquisition, path exclusions, and recovery behavior.
- It does not own the shared selection/snapshot record or cross-workstream handoff protocol; those require the shared coordinator contract.
- It does not perform engineering-style interpretation or change an adopter's canonical Project, policy, provider files, or code.
- Work may investigate in parallel on disjoint code/docs ownership. Shared selection records, target directories, and canonical consumer files require serialized coordination.

## Current implementation

`internal/app/init.go`, `init_plan.go`, and `init_write.go` build and apply a read-only preview with exclusive creation and revalidation. `cmd/markitect/init.go` exposes the current `--name`, `--namespace`, optional `--path`, and `--write` behavior. Tests are in `internal/app/init_test.go` and `cmd/markitect/init_test.go`. The current CLI requires a new Project and its target Area to be absent; there is no existing-project or root-inventory mode.

## Owned subsystem

For a future approved implementation, ownership is limited to new or directly relevant files in `internal/app/` for preparation and filesystem/Git safety, `cmd/markitect/` for an explicitly approved CLI surface, and focused tests for those components. The current Init path is a shared mutable hotspot: coordinate any changes to existing Init semantics with the shared coordinator. Documentation changes belong to the root-owned integration plan unless separately assigned.

## Allowed changes

- During investigation: read the current Init implementation, tests, usage contract, and filesystem/source snapshot helpers; produce a bounded design note or reviewable experiment in an assigned isolated work area.
- After contracts and implementation authorization: add a distinct existing-project preparation capability that requires explicit owner scope review, reports detected/selected/excluded roots, uses fixed inputs, and stages results in an approved isolated location.
- Add meaningful safety tests proving no writes before explicit apply intent, no overwrite/adoption of existing Project state, and safe handling of path aliases, exclusions, concurrent changes, and partial failure.
- Preserve the existing new-Project path, preview semantics, exclusive creates, and recovery reporting unless an explicit reviewed decision changes them.

## Forbidden changes

- Do not implement a feature now solely because this workstream exists; it is not implementation authorization.
- Do not reinterpret or remove current minimal Init's refusal of an existing `markitect.yaml` or existing target Area without an explicit coordinator decision.
- Do not invent a stable command name, flag set, selection schema, snapshot schema, workspace handoff, or persistent state format before approval.
- Do not inspect all repository contents implicitly, follow symlinks or nested repositories without an approved rule, broaden a selection through globs, or treat detected roots as selected roots.
- Do not rewrite, migrate, or activate consumer policy, adapters, package pins, provider files, AGENTS, docs, tests, or implementation code.
- Do not interpret source-code or repository conventions in the Markitect Core.

## Dependencies

- **Hard current behavior:** for existing-project preparation, preserve the current Init invariants until a separate behavior is explicitly approved. Current Init's required name/namespace, root safety, absent Project/Area paths, exclusive writes, and branch/lock checks remain the baseline for new-Project creation.
- **Hard future dependency:** coordinator-approved shared selection, snapshot identity, owner-review, and workspace-handoff contracts before durable implementation.
- **Soft choices:** default workspace location, root presentation, whether preparation can be previewed outside Git, and which optional project metadata is displayed. These remain design decisions; none creates permission to read unselected content.
- **External:** adopter-owned repository instructions and authority determine whether a consumer checkout may be inspected or modified. This workstream has no authority to alter those files.

## Design questions

- Is preparation a distinct command or a distinct Init mode? What exact user action marks the owner-reviewed scope, and how is that review bound to inputs?
- Which filesystem/Git roots can be detected safely? How are nested repositories, worktrees, submodules, ignored/generated paths, symlinks, and case aliases represented?
- Does a workspace contain only reports and candidate scaffolding, or does it ever materialize selected source bytes? Where does it live and how is it cleaned up or recovered?
- What exact immutable identity binds each root, and what changes invalidate the preparation plan between preview and write?
- How does preparation hand an approved selection to Copy Me without changing it or merging identities?
- Which existing Init guarantees must be repeated in the new mode, and which outcomes remain read-only by construction?

## Required tests

After design approval, cover at least:

- Existing `init` behavior remains byte-for-byte compatible for valid new-Project preview/write flows.
- An existing Project and existing Area are not overwritten, adopted, migrated, or partially rewritten by preparation.
- Detected roots, selected roots, and excluded roots remain distinct; excluded and unselected inputs do not enter later context or discovery.
- A changed branch, commit, manifest, directory identity, or selected bytes invalidates a saved preparation result before any write.
- Unsafe aliases, symlinks, nested repositories, ignored paths, staged targets, and partial writes follow the approved handling contract.
- No mutation occurs during inventory/preview, and explicit writes are confined to the approved workspace destination.

## Required evidence

Record the exact Markitect commit and build identity; owner-approved repository/root scope; detected, selected, and excluded roots with reasons; immutable source identity for each repository; selection and plan identities; relevant safety-test results; workspace output paths and hashes; and any partial-failure recovery evidence. Report uninspected roots and unresolved scope instead of describing them as covered. Do not retain credential values, secret contents, or unapproved personal information.

## Exit criteria

- The coordinator has approved shared selection, snapshot, and handoff contracts.
- A separate implementation decision identifies the exact Init behavior and allowed writes.
- Tests establish preservation of current Init behavior and the approved no-overwrite and fixed-input guarantees.
- Evidence shows the approved selection, excluded scope, destination, and generated artifacts are reviewable and reproducible.
- The root integration owner has reviewed compatibility and documentation changes before any release claim.

## Completion questions

- Did the owner review scope before any selected content was read or interpreted?
- Can a reviewer distinguish detected roots from selected and excluded roots and trace each to fixed identities?
- Did preparation preserve existing Project state and existing authorities?
- Is workspace handoff bound to the reviewed selection without silently broadening it?
- Are all writes confined to the approved workspace and covered by evidence?
