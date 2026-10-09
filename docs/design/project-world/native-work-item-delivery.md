# Native work item delivery

This implementation checklist closes the user workflow in [model-first-user-workflow.md](model-first-user-workflow.md), [operation-scopes-and-model-briefings.md](operation-scopes-and-model-briefings.md), and the [2026-10-09 completion mandate](../work-item-comparison-20261009.md). It is a delivery plan, not a validation claim.

The normal contributor gives their existing Codex or Claude agent a short work item. Product-installed project instructions and skills lead that agent through persistent exploration, canonical model editing, computed readiness, and the existing bounded Manager execution. The CLI supplies durable state and enforceable Host transitions; it does not replace the native conversation with another planning platform.

## Shared lifecycle

- Exploration records persist the request, named scopes, ideas, decisions, blocking questions, and proposed changes. Records and proposals are operational state, not accepted model declarations.
- New project manifests declare `workflowMode: guided` and `acceptancePolicy: committed-model`. Under this repository policy the selected committed canonical YAML is the accepted specification. A commit is provenance, not authenticated human approval; committing a draft never accepts that draft as canonical intent. Existing empty workflow mode remains a compatibility input.
- Readiness is recomputed for one named scope from a fixed accepted model, all required artifact paths and checks, selected Managers, exact runtime and briefing bindings, and resolved blocking decisions. A caller-authorized structure acknowledgement binds that exact selection. Previewing readiness starts no agents.
- Executable plans for guided projects require the corresponding exploration and readiness binding. The plan digest binds the receipt and immutable selection. Changed model, repository basis, scope, runtime, briefing, or acknowledgement invalidates it. No random plan identifier participates in the stable readiness selection, avoiding a circular receipt dependency.
- `deliver` composes Plan, Run or Resume, Verify, preflight, and guarded Apply. It retains the existing journals, failed attempts, cumulative resource ceilings, leaf implementer/reviewer cycles, parent integration, and full-repository final verification. Failures leave exploration active. Only a successful recorded Apply completes the scope; recovery reuses that receipt.
- Accepted-model history is captured from committed first-parent transitions. The first valid model is a baseline; later semantic changes and reversions each receive their own briefing. Code-only revisions advance the cursor without inventing model events. Missing or inconsistent history fails closed.
- Brownfield sessions bind separate fixed source and target bases and retain immutable discovery, scoped Manager proposals, parent integrations, explicit resolutions, model-only adoption plans, and receipts. Native agents work through that Manager tree. Unadopted scopes remain visible as transitional or unresolved; they are not evidence of conformance. Cleanup follows accepted adoption separately.

## Completion gates

| Boundary | Required validation |
| --- | --- |
| Explore/readiness | Closed durable records, CAS, scoped blockers, exact structure acknowledgement, stale basis/runtime, failed Apply remains active, successful Apply recovery |
| Brownfield | Source and target staleness, own evidence boundaries, parent integration, unresolved conflict preservation, immutable stages, model-only adoption, transitional coverage |
| Briefings | Initial baseline, adjacent semantic transitions, code-only commits, reversions, missing/shallow history, tampering and draft separation |
| Native contributor | Real generated Codex/Claude project files, ordinary work item entry, preserved existing configuration, idempotent installation, recovery guidance |
| Product | Executable Shop journey, finite local gates, independent review, bounded real-provider proof with every attempt retained, hosted checks, ordinary reviewed merge |

Historical evidence remains attached to its tested source revision. Passing protocol fixtures proves deterministic mechanics; provider usability, semantic correctness, human acceptance and comparative economic benefit are separate claims. No release is authorized by the main-promotion mandate.
