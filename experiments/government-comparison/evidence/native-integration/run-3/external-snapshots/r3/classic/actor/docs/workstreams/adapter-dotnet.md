# .NET adapter workstream

## Context

Start from the full verified preparation integration SHA supplied in the assignment, as defined by [Baseline](../development/baseline.md). PR 59's `e9550f5` is the source audit base only; it omits these preparation contracts and is not the dispatch baseline.

This preparation note describes the existing, separately built .NET reference adapter at the PR 59 integrated baseline (`e9550f5c91430c6a65cbbd4ffcdbb49b48272537`). It is not a feature request or claim that Markitect validates evaluated .NET build graphs. Shared command and semantic-model assumptions are owned by [the shared adapter contracts](../development/shared-contracts.md).

## Objective

Keep the reference adapter useful as a bounded, read-only example of an independently implemented command adapter. Preserve the exact limits of its ProjectReference evidence.

## Scope

The adapter compares explicitly mapped canonical `dependsOn` relationships with literal, unconditional `<ProjectReference Include="..." />` declarations in explicitly captured `.csproj` files. Work may clarify its mapping usability, evidence output, diagnostics, executable documentation, or isolated tests within that evidence boundary.

## Current implementation

The implementation is in `cmd/markitect-adapter-dotnet/`. It consumes a normalized model and staged inputs through the generic command protocol. It parses captured XML directly; it does not invoke MSBuild, evaluate SDK imports, inspect C# code, or mutate a repository. Unsupported conditions, imports, transformations, expressions, wildcards, and out-of-scope paths yield incomplete evidence. It is built separately and is not included in the native Markitect CLI release.

## Owned subsystem

The .NET adapter package, its README, adapter-specific fixtures, and tests under `cmd/markitect-adapter-dotnet/` are owned by this workstream. Its mapping and interpretation of `dependsOn` remain adapter-owned.

## Allowed changes

- Improve diagnostics or deterministic handling of literal XML cases without broadening claims.
- Add focused tests for explicit mappings, captured bytes, unsupported constructs, plan binding, and verification drift.
- Improve setup/configuration guidance in the adapter README.
- Propose a broader .NET evidence capability separately, with precise inputs and limitations, before changing its evidence contract.

## Forbidden changes

- Do not describe this as MSBuild evaluation, a complete project dependency graph, compilation verification, or source-code analysis.
- Do not parse arbitrary authoring YAML or use another adapter's output as canonical input.
- Do not add Core fields, change generic command semantics, or edit shared renderer/CLI files as part of an adapter-local change.
- Do not add mutation or apply behavior without a separately reviewed target, ownership, credentials, concurrency, and recovery design.

## Dependencies

Depends on the existing `semantic-model/v1alpha1` and command request/result/plan protocol, exact `config.inputs`, and a Project mapping for every in-scope resource/project file. Any changes to those shared contracts must be coordinated through [shared contracts](../development/shared-contracts.md). Other provider adapters are not dependencies.

## Design questions

- Is each mapped `.csproj` in scope and captured as an exact snapshot input?
- Are all reported edges literal and unconditional, and are unsupported cases visibly incomplete?
- Are unmapped resources intentionally outside the check scope?
- Does any proposed enhancement require evaluated MSBuild semantics? If so, define that as a new evidence capability instead of silently broadening this one.
- Is the executable's version/source provenance adequate for the consumer, given that Markitect's digest binds executable bytes but is not a signed provenance statement?

## Required tests

Run adapter package tests for literal mappings, missing/ambiguous mappings, forbidden and missing edges, unsupported XML semantics, and plan/verify drift. Retain the canonical-engineering integration test for protocol use. Run broader repository gates only when the eventual implementation change requires them.

## Required evidence

Record exact source revision, adapter executable digest, declared adapter version, input paths and digests, mapping scope, observation evidence kind, result status/findings, and the command used. State that this proves only literal captured XML correspondence. An executable-byte hash alone does not establish source provenance or publisher identity.

## Exit criteria

The adapter's configured mapping can be reviewed against exact canonical identity keys and captured project paths; its result distinguishes complete literal evidence from unsupported MSBuild semantics; focused tests pass; documentation makes no broader build-system claim.

## Completion questions

- Can a reviewer reproduce the result from the recorded model and captured files?
- Are all incomplete cases distinguishable from a verified match?
- Did the change stay within this adapter's files, or were shared-contract requests coordinated separately?
