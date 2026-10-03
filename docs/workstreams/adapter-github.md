# GitHub command adapter workstream

## Context

Start from the full verified preparation integration SHA supplied in the assignment, as defined by [Baseline](../development/baseline.md). PR 59's `e9550f5` is the source audit base only; it omits these preparation contracts and is not the dispatch baseline.

At the PR 59 integrated baseline (`e9550f5c91430c6a65cbbd4ffcdbb49b48272537`), Markitect has a generic external `command` adapter protocol but no built-in GitHub adapter. This is preparation for an independently developed consumer, not a decision to add a Core integration. Shared protocol ownership is described in [shared adapter contracts](../development/shared-contracts.md).

## Objective

Define a narrowly scoped GitHub observation adapter that maps selected canonical resources to explicitly identified GitHub objects and reports bounded evidence independently of other adapters.

## Scope

Adapter-owned executable, parameter mapping, isolated fixtures/tests and documentation. Choose specific GitHub objects and desired-state semantics before implementation. Start with read-only observe, then deterministic plan and verify using the existing generic protocol. Do not assume every resource is a repository, issue, project item, branch protection rule, or workflow.

## Current implementation

There is no GitHub-specific implementation or native adapter type. `spec.adapters` currently accepts `type: command`, exact captured inputs, plugin-owned parameters and argv for observe, plan and verify; apply is an explicit optional capability. The command runner consumes the normalized semantic-model DTO plus staged declared inputs and runs with caller authority. It does not register or discover adapters automatically.

## Owned subsystem

Future adapter code and tests belong in an isolated command/package such as `cmd/markitect-adapter-github/` (proposed location; it does not exist at this baseline), plus an exclusively owned fixture/workstream document. The GitHub mapping and interpretation are adapter-owned. Generic runner/protocol changes are not.

## Allowed changes

- Define explicit mappings from canonical identities to exact GitHub object identities without assuming universal Markitect kinds.
- Implement a read-only observer for one explicit object family and a limited, documented evidence claim.
- Produce deterministic plan/verify results bound to the supplied model, mapping, observation and target.
- Use adapter-owned secure credential lookup; retain only non-secret target/mapping data in Project configuration and plans.
- Demonstrate target partitioning so parallel adapters or runs do not compete for writes.

## Forbidden changes

- Do not add GitHub fields or semantics to Markitect Core or infer resources from repository names, paths, or prose.
- Do not call or mutate GitHub from built-in renderer, `check`, `model`, or other adapter executables.
- Do not treat another adapter's output as canonical desired state.
- Do not put credentials/tokens in Project YAML, logs, results, or saved plans.
- Do not enable shared-target apply, retries, or concurrent mutation before target ownership, concurrency, idempotency and partial-failure recovery are designed.
- Do not change shared command contracts without a coordinated contract decision.

## Dependencies

Depends on the current versioned command/model protocol and explicit canonical mappings. The exact authentication mechanism, API version/permissions, object family, retry/rate-limit behavior, and target ownership are undecided external design inputs. Follow [shared adapter contracts](../development/shared-contracts.md); no other adapter is a dependency.

## Design questions

- What exact GitHub objects are observed and which fields constitute desired state?
- Which canonical kinds/resources are in scope for this adopter, and who owns each mapping?
- What non-secret target key uniquely identifies repository, organization, environment and object scope?
- Can read, plan and verify be guaranteed free of writes by implementation and tests?
- How are credentials resolved securely, permission-bounded, rotated and prevented from entering evidence?
- If apply is later proposed, what serializes concurrent runs, protects against human/API drift, handles rate limits and recovers partial multi-object updates?
- What is incomplete versus failed, and what can observation not establish?

## Required tests

Before any implementation, define testable object semantics. Then test mapping validation, target echo/identity, independent read-only observe, deterministic plan over captured observation, verify drift, missing/permission-limited objects, API error/rate-limit handling, and secret redaction. Keep tests isolated from production accounts; use fixtures or a controlled test environment. Apply tests require a separately approved disposable target and concurrency/recovery scenarios.

## Required evidence

Record exact Markitect source revision and protocol versions, adapter source/binary digest, declared adapter version, non-secret target identity, mapping scope, API/tool version, observed object identifiers and selected fields, timestamps/observation digest, status/findings, and permission/error limits. An executable-byte digest is not signed source provenance. Never retain credentials in output.

## Exit criteria

The adapter has one explicit object family and mapping contract, read-only behavior is demonstrated, observation and plan evidence are reproducible and target-scoped, secrets are absent from configuration/results, tests pass, and no Core or shared renderer edits were needed. Any write/apply expansion remains a separate gate.

## Completion questions

- Does the adapter derive desired meaning only from the normalized model and its own explicit mapping?
- Is every external read and result bound to a stable target identity and observation?
- Can two configured adapters be shown to have disjoint target scopes?
- Were credentials, concurrent writers, API drift, and partial failures handled within the stated scope?
