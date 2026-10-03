# First parallel development wave

Status: dispatched; integration is pending. Published v0.12.0 remains unchanged. This record measures subsystem independence, not productivity or adopter acceptance.

## Fixed baseline and gates

All workstreams start from `0ba7a7218f2ceae65b8db90bb8133c2ffa583ada`, the PR 60 preparation merge. [Main CI 37144006754](https://github.com/Glacius-Labs/Markitect/actions/runs/37144006754) passed Windows, Linux and source-artifact gates at that exact commit; verified again before dispatch. The [workstream specifications](../workstreams/README.md), [shared contracts](../development/shared-contracts.md) and [constitution](../engineering-constitution.md) are implementer authority. Earlier proposals remain design inputs.

## Assignments and live integration record

The explicit assignments below narrow the broader future-work permissions in the workstream specifications for this wave; a future specification is not authority to exceed the assigned paths. GitHub/Azure consume offline captures only: no live fetch, credential lookup or credential use.

Each branch has a managed isolated worktree; shared renderer files, CLI/schema/semantic contracts, dependencies, public product documentation and integration remain coordinator-owned. Implementers commit local candidates; the coordinator serializes shared-ref publication and merges. Every source candidate requires exact-head Windows/Linux CI and independent full-diff review.

| Workstream | Branch | Bounded first-wave deliverable | Status / dependency | PR / CI / integration |
|---|---|---|---|---|
| Init | `codex/wave-init` | Explicit scope and isolated preparation design; implementation requires reviewed handoff | Proposal complete; durable implementation deferred pending selective-acquisition/storage decision | Review pending |
| Copy Me | `codex/wave-copy` | Evidence/candidate review design; binding helper requires reviewed handoff | Proposal complete; durable implementation deferred with Init shared handoff | Review pending |
| Risk | `codex/wave-risk` | Risk classifications and command-plan negative controls | Independent tests/report | Pending |
| .NET | `codex/wave-dotnet` | Literal ProjectReference evidence audit and bounded hardening | Independent owned command | Pending |
| Docs | `codex/wave-docs` | Projection ownership and human-content negative controls | Disjoint new tests; shared production changes require request | Pending |
| Codex | `codex/wave-codex` | Provider projection independence negative controls | Disjoint new tests; same shared-file gate | Pending |
| Claude | `codex/wave-claude` | Provider projection independence negative controls | Disjoint new tests; same shared-file gate | Pending |
| GitHub | `codex/wave-github` | One read-only repository-metadata conformance consumer | Offline fullName/defaultBranch mapping approved; implementing | Pending |
| Azure DevOps | `codex/wave-azure` | One read-only repository-metadata conformance consumer | Offline defaultBranch mapping approved; implementing | Pending |
| Konfyra | `codex/wave-konfyra` | Read-only consumer inventory; sanitized product gaps only | Consumer authority/privacy review before selected reads | Pending |

## Frozen boundaries and requests

- No new Core primitive, provider-specific semantic field, query capability, inheritance or implicit activation.
- External adapters consume canonical normalized meaning and explicit captured inputs. Captured metadata is observed evidence, never proof of current remote state. No credentials, live provider writes or Apply capability in this wave.
- Native Docs/Codex/Claude have independent goals but share renderer implementation. Their initial ownership is new, separate tests/reports; concrete production defects return to the coordinator.
- Init prepares selected evidence; Copy Me interprets and proposes; owner review and normal canonical adoption remain separate. Neither may independently publish a new shared record/CLI contract.
- Konfyra remains a consumer. Private source material, credentials and customer data must not enter this repository. Inventory does not authorize adoption or replace existing guidance.
- Historical pilot evidence, package versions and published release artifacts remain immutable.

## Coordinator decisions during execution

- **Native assignment scope:** future workstream specifications allow broader provider work; this wave explicitly assigns only disjoint new tests/reports. Production changes require a concrete request and coordinator decision.
- **Remote-provider scope:** GitHub compares canonical repository identity/defaultBranch with a supplied sanitized repository capture; Azure compares canonical defaultBranch with supplied captures mapped to exact organization/project/repository identities. Both are offline evidence consumers, with operation-free Plans and no Apply. Neither proves current provider state or authenticates the capture. Desired values come from normalized resources, not duplicated expected-value configuration.
- **Init/Copy Me:** proposal-only first-stage deliverables are complete. Existing source.Load acquires whole-repository bytes, so filtering it afterwards cannot prove selection privacy. Durable implementation is deferred: review selective exact-blob acquisition, external workspace storage/retention, and the minimal common handoff together before establishing a CLI or persistent record. Existing Init and discovery Workflow remain unchanged. This is a bounded design decision, not a new Core requirement.
- **Execution environment:** default-sandbox shell calls stalled even on Get-Location. Narrow authorized repository operations with login disabled and escalation returned promptly. This is a tooling fan-out limitation, not evidence of adapter semantic coupling.

## Completion evidence

Pending. Record reviewed heads, integration commits, gates, shared requests, concrete negative findings, deferred scope and answers to the sixteen coordinator questions before closing the wave. A successful wave is not a release decision.
