# Azure DevOps command adapter workstream

## Context

Start from the full verified preparation integration SHA supplied in the assignment, as defined by [Baseline](../development/baseline.md). PR 59's `e9550f5` is the source audit base only; it omits these preparation contracts and is not the dispatch baseline.

At the PR 59 integrated baseline (`e9550f5c91430c6a65cbbd4ffcdbb49b48272537`), Markitect has a generic external `command` adapter protocol but no Azure DevOps-specific adapter. This note prepares an independent consumer workstream; it does not add Azure concepts to the Markitect model. Shared protocol ownership is described in [shared adapter contracts](../development/shared-contracts.md).

## Objective

Define a narrowly scoped Azure DevOps observation adapter that maps selected canonical resources to explicitly identified Azure DevOps objects and reports bounded evidence independently of other adapters.

## Scope

Adapter-owned executable, mapping parameters, isolated tests/fixtures and documentation. Choose specific Azure DevOps object types and desired-state semantics before implementation. Start with read-only observe; implement deterministic plan and verify against that observation using the existing protocol. Do not assume universal Markitect kinds or that every organization uses the same projects, area paths, work items, pipelines, policies or repositories.

## Current implementation

There is no Azure DevOps-specific implementation or native adapter type. Project adapter configuration supports `type: command`, exact snapshot inputs, adapter-owned parameters, and required observe/plan/verify argv; apply is optional and explicitly gated. The generic runner provides normalized model input, staged files, bounded execution, protocol validation and plan binding. It does not supply Azure identity, authentication, target locking or a plugin registry.

## Owned subsystem

Future adapter implementation and tests belong in an isolated package such as `cmd/markitect-adapter-azure-devops/` (proposed location; it does not exist at this baseline), along with its separate fixtures and documentation. The adapter owns its Azure object mapping and evidence interpretation. Shared Markitect runner, Core and renderer remain separately owned.

## Allowed changes

- Define explicit adopter-owned mappings to stable Azure DevOps organization/project/object identities.
- Implement read-only observation for one declared object family and documented fields.
- Produce deterministic plan and verify results bound to current model/config/observation/target.
- Resolve credentials through adapter-owned secure mechanisms; store only non-secret identifiers and mappings in Project YAML and plans.
- Partition target ownership explicitly if more than one adapter or run may operate in the same organization.

## Forbidden changes

- Do not add Azure-specific fields, identities, taxonomies or policy to generic Markitect Core.
- Do not infer work-item meanings, area ownership, project identities or release status from resource names or prose.
- Do not route through, consume, or mutate another adapter's generated representation.
- Do not store PATs, bearer tokens, client secrets, or other credentials in config, logs, result evidence or plans.
- Do not enable apply to shared Azure objects until object ownership, concurrent runs, human edits, version conflicts, retry behavior and partial-failure recovery are designed.
- Do not edit shared protocol, CLI, or native renderer files for Azure-specific behavior.

## Dependencies

Depends on the current command/model protocol and explicit resource mappings. Organization/project topology, object family, credential source, permissions, API version, eventual-consistency behavior, concurrency mechanism and target identity remain design inputs. See [shared adapter contracts](../development/shared-contracts.md). No GitHub, .NET or native renderer adapter is a dependency.

## Design questions

- Which exact Azure DevOps API object family is in scope, and what is the precise desired-state comparison?
- Which canonical resources map to those objects for this adopter? Who owns and reviews those mappings?
- What non-secret target identity names organization, project, environment and object scope unambiguously?
- Can observe/plan/verify be shown to be read-only and repeatable despite eventual consistency?
- How are credentials least-privileged, acquired outside Project data, rotated, and redacted?
- If apply is later requested, what prevents two adapters/runs or a human edit from racing? How are ETags/revisions, rate limits, retries and partial multi-object updates handled?
- Which API conditions are incomplete evidence versus hard failure, and what does a completed observation not prove?

## Required tests

Define exact object semantics first. Then test mapping/target validation, read-only observation, deterministic plan from a captured observation, verify drift and stale plans, missing/unauthorized objects, API version/errors, retry/rate-limit semantics, eventual-consistency bounds, and secret redaction. Use fixtures or a dedicated disposable organization/project, never production mutation for tests. Apply tests require a separately approved isolated target plus concurrency and recovery cases.

## Required evidence

Record Markitect source revision and protocol versions, adapter source/binary digest, declared version, non-secret target identity, canonical mapping scope, Azure API/tool version, observed object IDs and selected values, timestamps/observation digest, result status/findings, and permission/consistency limits. The binary hash detects byte changes but does not prove signed provenance. Do not retain credentials.

## Exit criteria

The adapter has one explicit object-family contract and adopter-owned mapping, read-only behavior and evidence scope are demonstrated, results bind to reproducible observations and targets, secrets are absent from artifacts, tests pass, and implementation requires no Core or other-adapter changes. Apply remains a separate design and approval gate.

## Completion questions

- Does the adapter consume canonical normalized meaning plus only its declared captured inputs?
- Is each observation bound to an exact, non-secret organization/project/object target?
- Are duplicate/overlapping target scopes prevented by explicit ownership coordination?
- Are permissions, credentials, concurrency, drift, API errors, eventual consistency and partial failure bounded honestly?
