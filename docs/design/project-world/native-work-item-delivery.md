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

## Requirement to implementation map

| Requirement | Source owner and executable evidence |
| --- | --- |
| Durable exploration and scoped readiness | `projectexplore` records, CAS storage and readiness; `projectcli/project_explore_test.go` exercises runtime-free CRUD and exact acknowledgement preview/write |
| Ordinary work item and resume | `projectrun/deliver.go` composes the existing journals and guarded transitions; `deliver_test.go` covers stale bases, interrupted stages, retained failures and successful Apply completion |
| Automatic committed-model history | `projectbriefing` first-parent history and immutable bundles; `history_test.go` covers baselines, semantic changes, reversions, drafts and inconsistent history |
| Recursive Brownfield organization | `projectadoption/session.go`, `reverse.go` and `manager_integration_context.go`; `session_test.go` covers selected evidence, duplicate-child rejection, public child reports and exact integration bindings |
| Actual Brownfield Manager invocation | `projectadoption/manager_run*.go` and `projectcli/project_brownfield_run.go` reuse `agentexec`; focused tests use controlled invokers for preview pins, retained attempts, limits and report validation. These are transport tests, not real-provider evidence |
| Model-only adoption | `projectadoption/session_apply.go` retains an actual guarded Apply receipt; the tests reject external fabricated receipts and stale/replaced proposals |
| Native Codex and Claude entry | `projectonboarding/render.go` installs discoverable skills and project instructions; onboarding tests cover preservation, idempotence and Brownfield stage guidance |
| Whole-project realization | `projectrun` retains full Manager verification, explicit file ownership, checks and one guarded candidate; operations tests cover unimpacted siblings and unclassified files |
| Executable public example | `examples/project_world_workflow_test.go` exercises an isolated Shop checkout; Python application tests cover finite cancellation and transaction cases |

This map states where a contract is implemented and tested. Actual execution status is recorded in [native-delivery validation](../../validation/project-native-delivery-2026-10-09.md), including failed attempts and checks that have not run on the final candidate.

## Separate authoring assessments

The roadmap's [knowledge-graph and Markdown-frontmatter questions](../../implementation-plan.md#knowledge-graph-and-markdown-authoring-assessment) remain deferred assessments. This delivery package retains YAML resources as the sole canonical typed model and generated Markdown as the readable view. It does not add RDF/SHACL/SKOS/OWL inference or migrate canonical resources into Markdown. The requested semantic-question analysis and same-content authoring comparison have not been completed; this is not a rejection based on a completed experiment.

Historical evidence remains attached to its tested source revision. Passing protocol fixtures proves deterministic mechanics; provider usability, semantic correctness, human acceptance and comparative economic benefit are separate claims. No release is authorized by the main-promotion mandate.
