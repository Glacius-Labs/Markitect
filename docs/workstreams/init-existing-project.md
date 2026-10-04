# Existing-project preparation workstream

**Status:** `prepare` and the shared handoff are implemented in current source outside Core; source verification is in progress and this is not a CI or release claim. Published v0.12.0 remains unchanged. This page retains the original workstream rationale; the current contract is [Selective adoption handoff](../design/selective-adoption-handoff.md), with current CLI usage linked from [Usage](../usage.md) when integrated. See also [shared contracts](../development/shared-contracts.md) and [adoption boundaries](../development/adoption-boundaries.md).

## Context

Published v0.12.0 `init` remains the greenfield contract. Current source adds a separate `prepare` command to capture an explicitly owner-selected evidence scope into an external handoff workspace; it does not extend `init` or load/adopt an existing Project. The shared contract documents its fixed-input boundary.

Published v0.12.0 `init` starts a new Project only when `markitect.yaml` is absent and the target Area does not exist. It previews and can create only `markitect.yaml` plus one Area README. Its refusal to replace or adopt an existing Project remains a safety invariant. The source-only `prepare` workflow is separate and does not load or mutate an existing Project.

The implemented preparation requires the owner to supply exact roots, revisions, selected paths, reasons and exclusions; it does not infer or inventory candidate roots. The shared record, identity, write and recovery semantics are specified in the linked design. Owner/reviewer fields are supplied claims, not authenticated authorization.

## Objective

Maintain the separate preparation operation for an existing repository using an owner-supplied exact scope and isolated handoff. Preserve the existing greenfield Init behavior and refusal boundary. Any expansion beyond the implemented selection/capture flow requires a separate decision.

## Scope

- This workstream owns safe preparation and workspace creation, implemented separately from greenfield Init.
- Shared scope, snapshot, handoff and record semantics are centralized in the coordinator-owned [selective adoption contract](../design/selective-adoption-handoff.md).
- It does not perform engineering-style interpretation or change an adopter's canonical Project, policy, provider files, or code.
- Work may investigate in parallel on disjoint code/docs ownership. Shared selection records, target directories, and canonical consumer files require serialized coordination.

## Current implementation

Greenfield `init` remains implemented by `internal/app/init.go`, `init_plan.go`, `init_write.go`, and `cmd/markitect/init.go`; it still requires absent Project and Area paths. Current source adds a separate `prepare` command through `cmd/markitect/adoption.go`, `internal/app/adoption_prepare.go`, `internal/adoption`, and `internal/source/selective.go`. Exact CLI semantics and verification status are documented centrally rather than repeated here.

## Owned subsystem

The separate implementation owns its new app/source/record/CLI packages. Existing Init behavior, Core/Project contracts, source snapshot semantics, CLI dispatch, integration and release remain coordinator-reviewed shared areas.

## Allowed changes

- Preserve explicit scope, exact fixed inputs, exclusive external workspace creation, and existing Init behavior.
- Keep owner-supplied privacy/approval fields as claims; do not add authentication or expand scope from discovered repository contents.
- Treat source validation and release/published status as separate gates; this workstream page does not imply either has passed.

## Forbidden changes

- Do not treat this historical workstream specification as the current CLI or record-format reference; use the linked design and Usage.
- Do not reinterpret or remove current minimal Init's refusal of an existing `markitect.yaml` or existing target Area.
- Do not add implicit discovery, consumer repository writes, Core changes, or automatic adoption.
- Do not inspect all repository contents implicitly, follow symlinks or nested repositories without an approved rule, broaden a selection through globs, or treat detected roots as selected roots.
- Do not rewrite, migrate, or activate consumer policy, adapters, package pins, provider files, AGENTS, docs, tests, or implementation code.
- Do not interpret source-code or repository conventions in the Markitect Core.

## Dependencies

- **Hard current behavior:** for existing-project preparation, preserve the current Init invariants until a separate behavior is explicitly approved. Current Init's required name/namespace, root safety, absent Project/Area paths, exclusive writes, and branch/lock checks remain the baseline for new-Project creation.
- **Implemented source boundary:** shared selection, snapshot identity, owner-scope claims, and external workspace handoff are defined in the coordinator-owned design.
- **Out of scope for this source slice:** automatic root inventory, workspace defaults, or Project metadata inspection. The caller names an external destination and supplies the exact evidence scope; optional presentation changes cannot broaden that scope.
- **External:** adopter-owned repository instructions and authority determine whether a consumer checkout may be inspected or modified. This workstream has no authority to alter those files.

## Questions preserved from the proposal

- Preparation is a separate command, not a mode that changes greenfield Init. The handoff contract specifies its supplied scope and fixed-input boundary.
- The current contract does not discover roots; unsupported paths/repository shapes fail closed as described in the linked design.
- The external workspace contains the handoff and selected evidence bytes; exact identity/digest and failure behavior are defined in the linked contract.
- Copy Me consumes that handoff without merging repository identities or changing scope.
- Greenfield Init keeps its existing safety guarantees; preparation uses the separate external-write contract and never adopts a Project.

## Verification expectations

Candidate validation should show that exact selection/capture and external handoff boundaries follow the [canonical design](../design/selective-adoption-handoff.md), and that greenfield `init` regression behavior remains intact. Do not claim detected-root inventory: this implementation does not include it. Full local/CI results belong to immutable candidate evidence, not this specification.

## Required evidence

For a verified candidate, report the exact Markitect commit/build; supplied scope and exclusions; fixed source identities; selection, capture and handoff digests; safety-test results; external workspace paths/hashes; and partial-failure behavior. Report uninspected evidence honestly and do not retain credential values, secrets, or unapproved personal information.

## Exit criteria

- Focused and required repository/CI gates pass for an exact candidate SHA.
- Review confirms greenfield Init remains unchanged and the new workspace boundary matches the shared contract.
- The root integration owner records the candidate result; a separate immutable publication record is required for any release claim.

## Completion questions

- Did the owner review scope before any selected content was read or interpreted?
- Can a reviewer distinguish detected roots from selected and excluded roots and trace each to fixed identities?
- Did preparation preserve existing Project state and existing authorities?
- Is workspace handoff bound to the reviewed selection without silently broadening it?
- Are all writes confined to the approved workspace and covered by evidence?
