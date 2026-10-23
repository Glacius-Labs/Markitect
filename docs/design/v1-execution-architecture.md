# V1 execution architecture

Corrected v0.17 architecture, accepted for source implementation on 23 October 2026 in the [owner approval](../concepts/sources/v1-source-implementation-approval-20261023.md). Real evidence follows a concrete bounded coordinator dispatch under the subsequent standing owner authorization. The [latest owner clarification](../concepts/sources/v1-scope-clarification-20261023.md) overrides conflicting v0.16 preservation and hierarchy rules. The [counter-review](../development/v1-architecture-audit-20261023.md) binds observations to source; the [delivery plan](../implementation-plan.md) owns sequencing. These current documents replace the previous F/H/G/E/N/P/L assignments.

## Scope

Build Model -> Artifacts. Preserve canonical language, identities, imports, compiler diagnostics, registry, ownership, accounting and suitable deterministic foundations. Existing runtime APIs, implementations and package boundaries have no preservation right. Extract, Cockpit/UI, interactive authoring and their migration impose no requirements on the new foundation. Their active consumers are retired in the bounded cut, not ported.

Assessment, Implement and Verify each use the same complete model-derived organization: Root and every local Manager, including empty and intermediate Packages. Root-only is valid only for an actually Root-only model. One executor handles all mandates. Review mandates aggregate judgments; their scratch changes are not candidate product changes. Run organizes Managers; workflow chooses Runs; the command publishes.

## Package direction

These are concrete proposed locations, not preservation obligations for existing code.

| Package | Responsibility and reason | Inward dependencies |
| --- | --- | --- |
| `internal/model`, `internal/snapshot` | Preserved canonical checking and immutable source values | Existing pure dependencies |
| `internal/organization`, `internal/realization` | Complete topology/context and registry/ownership/accounting | Model, snapshot; no executor |
| `internal/project` (new) | Checked capture/preparation shared by check/context/docs, standalone Run and Apply | Foundations and its own source port; no Apply/Run |
| `internal/harness` (new) | Pure versioned rules, project/Manager context and mandate instructions consumed by native renderers in the two provider adapters | Project/context values; no Run, provider or I/O |
| `internal/run` (replacement) | Sole public Execute lifecycle, hierarchy, messages, handoffs, feedback, durable result and cleanup | Project, harness, foundations, ports consumed here |
| `internal/workflow` (new) | One direct Apply sequence; separate import gate prevents Git/publication/provider logic entering it | Run values and its own Executor interface only |
| `internal/apply` (replacement) | Command preflight, shared allowance/deadline, workflow invocation and final publication | Project, workflow, Run result values, its own publication port |
| `internal/adapters/gitrepo` | Source capture, Run copies/integration, durable Git bundles, guarded publication in separate collaborators | Consumer ports in project/run/apply, snapshot, technical transport |
| `internal/adapters/runstore` (new) | One file-backed adapter for terminal results and separate project Notes | Run values; no orchestration |
| Codex/Claude adapters | Native harness rendering/installation/use, normal account login and technical sessions | Run session contract, harness, container/agentd |
| Container, agentd, check adapter | Process/resource transport and isolated project checks | Technical or Run check contracts; no workflow |
| Bootstrap and CLI | Composition, parsing and result display | Actual application services; no second lifecycle |
| Modeldocs/documents adapter | Existing readable canonical view and guarded write | Rebind preparation to project, model, snapshot; no executor |

CLI -> Apply -> Workflow -> Run -> Project/foundations. Diagnostic commands call Project directly. Adapters are injected by Bootstrap. Run cannot import Apply/Workflow/adapters; foundation cannot import execution. Enforce directions for tests too. Four new package roots are justified by actual consumers, not a numerical limit. No shared-contracts mega-package, repository framework, DSL, event bus or duplicate model ontology.

## Minimal model connection

Project capture invokes existing `model.Check`, registry parsing/accounting, `organization.BuildTopology` and `SelectContexts` on fixed bytes. Reuse their values/diagnostics directly. Preserve source closure, dependency/missing-input diagnostics, empty Packages, explicit ignore classifications and exact captures. Check must not need credentials, images or container metadata.

`project.Prepared` holds the checked input capture and manifest, canonical result, topology/context, registry/layout and declared checks. It is an immutable in-process value, not another serialized compiler AST. Persistence retains sufficient fixed source/input identities to recapture and verify it. Model content digest alone does not cover the complete organizational Package inventory. Git commit, tree and model identity remain distinct.

Do not inherit Extract flags, artifact-advisory policies, PassSpec, strategy/review-team settings or legacy preparation DTOs. Runtime profile decoding belongs outside the compiler. Apply fixes canonical/control/ignore inputs; valid registry updates remain realization work. Candidate validation reuses foundation logic against fixed intent and candidate registry/tree, not a second model validator.

## Necessary contracts with actual consumers

Land each contract with a caller and concrete implementation, never an interface-only wave.

| Contract owner | Producer -> consumer | Contents |
| --- | --- | --- |
| Project: Prepared/CandidateRef | Capture/Git store -> Run, diagnostics, Apply | Checked input binding; durable project-scoped storage key, Git object format, commit/tree; no live object/volume |
| Run: Request/Result | Workflow or standalone caller -> Execute -> workflow/CLI | ID, mandate, prepared binding, input candidate, required Findings and shared allowance; technical completion, judgment, output reference, observations, actual cleanup |
| Workflow: Executor | Run -> direct Apply composition | One fully closed Run per call; workflow chooses subsequent call |
| Run: Sessions/session | Native adapter -> executor | Open with harness/workspace, ordinary messages/tools, cancellation/closure and observations; no domain Turn/Pass |
| Run: workspace/candidate port | Git adapter -> executor | Open one Run, task bases, fixed handoffs, reviewed integration, retain/reopen candidate and close; handles stay inside execution |
| Run: records/Notes port | Runstore -> executor/Manager tools | Save/read closed record; append/read separate project Notes with origin/context |
| Run: checks port | Check adapter -> executor | Named check on exact candidate, actual exit/timeout/cleanup observations |
| Apply: publication port | Git adapter -> command | Durable accepted candidate, source/target guards, transfer/readback; no living Run |

The Run integration owner alone defines and changes the consumer-local Sessions/session port. The Harness/Provider owner implements it and supplies capability evidence; a joint reviewed handoff precedes contract freeze. Provider-dependent signature decisions stay provisional until the early native gate below. Use consumer-local interfaces. One concrete adapter can implement several small ports; no combined Git/source/publication interface. No session handle appears in persistent results. Technical completed/failed/incomplete is separate from semantic approve/reject/model blocker. Implement submits a checked candidate, not publication approval. Lost transport is not Reject. Host binds known identities; a small explicit tool records actual semantic judgment, without mandatory self-reported commit IDs or ritual completion JSON.

## Complete Run lifecycle

`run.Executor.Execute` is the single closure owner, including standalone use:

1. Validate checked binding, full topology, mandate and finite allowance. Open one Git Workspace Service for this Run and fresh Manager environments only for its topology. No future-Run reservations. Each Manager retains its copy/conversation within this Run.
2. Install harness and coordinate the full hierarchy. Deliver questions even while other Managers wait; serialize mutations to one parent's copy, not all children. Cancellation releases waiters. Parent reviews/integrates fixed child results in completion order.
3. End active writes before capturing a handoff/Root commit, using ordinary session synchronization and internal Git commits. Bind decisions/checks to that immutable object; a later change requires renewed affected checks/review. No per-write/freeze protocol.
4. Persist required candidate objects and verify independent readability before deleting temporary copies. Retain diagnostic partial candidates where possible; failed retention never yields acceptance.
5. Close every Manager process/container/home, Run Git service and temporary volumes, including partial-start failures. Use bounded cleanup even after work cancellation; no further agent work is granted. Record actual close errors.
6. Save and read back the terminal result including cleanup outcome, then return. Storage failure returns technical error plus available diagnostic identities; never claim a saved result exists. A crash before terminal storage is incomplete, not inferred success or automatic resume.

Cleanup uncertainty, missing decisions and persistence failures block successful return/publication. Standalone caller provides finite limits. Apply supplies one shared concurrency-safe allowance and original deadline to every Run, including corrections. Resource counts have explicit technical meaning, not domain Turn semantics; no reset after Reject.

## Candidates, state and publication

Propose self-contained Git bundles with a named candidate ref and bound metadata in project-scoped operator state outside the repository. Include candidate objects and required ancestry without an old-volume or incremental-bundle dependency. Verify in an empty Git repository with exact commit/tree readback. A bundle is storage, not approval ([official Git documentation](https://git-scm.com/docs/git-bundle)). No new Git server or recovery framework.

Git objects and Runstore use the same project state root with separate records. Secure candidate data before writing its terminal reference. Terminal records use temporary-file/atomic publication, appropriate flush/readback and tested platform failure handling; do not claim a multi-file transaction. Notes use unique append records with atomic visibility and bounded reads so concurrent processes cannot overwrite them. No database/event-sourcing or automatic retained-evidence garbage collection is required for initial V1.

Apply waits for every preceding Run to return, then reopens persisted acceptance and candidate data. Approval binds exact candidate and project inputs. Publication uses host Git in an isolated temporary repository, never the closed Run service; no Agent container remains. Recheck candidate and target/source guards before transfer. Any relevant drift from the captured clean target ends in conflict for narrow V1; no semantic merge or approval reuse for changed content. No automatic owner commit/push.

Check guards immediately before mutation and read back afterward. Cooperating command locks plus guarded per-path transfer limit races; they cannot make an unrelated editor transactionally exclusive. Preserve intervening user changes and report partial transfer/readback failure honestly, rather than promise global atomicity.

## Checks and Manager responsibility

The same deterministic core serves four checkpoints: command preflight before resources, every fixed child handoff, fixed Root completion, and publication. Handoff/Root problems yield repair feedback within the Run first. Unresolved invalidity blocks Implement success/approval. Review may successfully reject an invalid candidate; it must not repair that candidate just to obtain a green gate. Required project tests remain meaningfully scoped and commit-bound; not every child runs the entire suite.

Accounting proves registered ownership, not who changed a file. Review additionally receives Manager/task, task-base/result diff, and exact accepted child commits/integration bases. Check unexplained out-of-area changes, while recognizing legitimate child integration. Never subtract accepted child paths/hunks from the parent diff: later parent edits could disappear. Parent conflict-resolution changes still require review; unresolved child-owned work returns to the child. Keep small Git review context, not a global provenance ledger or ACL system.

## Harness, provider and authentication

One Harness/Provider owner covers the complete chain: versioned Markitect rules -> canonical project -> Manager ownership/context -> mandate/Findings -> native files/tools/start -> actual provider loading. Pure text/digest tests are insufficient. Generic AGENTS content is a representation fallback, not an invented runtime adapter. Generated instructions/tools live outside product candidate content. Mandatory-context overflow is explicit, never silent truncation.

Inventory repository instruction files and native load precedence. Use supported native overrides/controlled instruction mounts to make canonical generated instructions authoritative without editing tracked files. If a pinned provider cannot prevent contradictory instruction loading, fail that capability gate. Test actual loaded paths, tool availability and priority per provider. Native subagents do not replace modeled Managers.

Preferred auth design: start the Manager container with agentd awaiting initialization, provision its private home through one bounded private setup message, then start the provider. This requires a new narrow agentd/container capability; today's session API does not provide it. Allowlist auth paths, prevent symlink traversal, use owner-only permissions, redact secrets, exclude secrets from argv/env/logs/harness/Notes/bundles and clean partial setup. Validate expiry/refresh semantics per provider; never share a mutable host login home or blindly copy refresh authority. Prepare the finite native probe during S1. At the earliest usable S2 transport checkpoint, run it under a concrete bounded coordinator dispatch, before freezing provider-dependent session/setup signatures or integrating adapters that rely on them. Without an appropriate bounded dispatch or successful evidence, those decisions and adapter integration remain blocked; continue only independent deterministic work. This is a hard dependency gate, not a late S4 observation. Feasibility testing must establish safe normal account login or report the concrete blocker. No renamed auth-helper container.

Remove Codex API-key resolution, profile/config branches, adapter route, examples and branch-specific tests together. Preserve secret validation/redaction tests for account login. Native `turn/start` and Claude conversation identifiers remain adapter facts. Codex and Claude qualification is independent; one does not certify the other.

Notes are persistent project-scoped noncanonical hints with provenance; they do not grant authority or schedule work. Mandatory Findings stay in results and explicit next-Run inputs. Verify sees fixed model/candidate and necessary technical observations, not Implement conversations/self-approval. Notes remain attributed hints. Hidden outside-oracle expectations never enter Implement context.

## Acceptance and decisions

First prove a complete small hierarchy using real local Git, controlled sessions, durable result/Notes, cleanup, reopening and publication; then all-mandate freshness and failure branches. Prepare a separately authorized early native probe before broad dependent work. Final qualification covers complete source gates, preservation/no-change, seeded defects and an outside oracle, plus each claimed provider separately.

No unresolved product choice blocks planning: hierarchy, exclusions, Notes and normal login are settled. The corrected plan has explicit source-implementation approval. The subsequent owner instruction delegates useful finite real proofs to the coordinator; claimed release provider support still follows actual evidence. Routine API, package or storage choices do not create extra permission rounds.
