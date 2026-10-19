# Delivery plan: the first reliable Apply

> **Later accepted direction, 19 October 2026:** [DEC-058](concepts/register.md#dec-058-apply-always-executes-before-independent-verify) amends the entry order: every Apply starts with Execute and integration at Root, proceeds through the fixed tree and then runs independent Verify. No initial Verify and no special route for absent artifacts. The [current concept](design/canonical-model/manager-apply-concept.md#shared-working-tree-and-reconciliation-cycle) owns this target; the runtime contract and runnable self-model still describe the previous implemented order. This documentation-only instruction authorizes no runtime/test changes, model migration or provider run.

> **Owner direction, 19 October 2026:** [DEC-056](concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace) and the [current concept / later Claude handoff](design/canonical-model/manager-apply-concept.md#current-agreed-concept--19-october-2026) replace the conflicting sequential/private-workspace and delivery-review process below. This change documents intent only. Existing implementation assignments remain bounded to their sessions; the historical queue below is not a new dispatch or authority to interfere with active writers. Reconcile concrete delivery assignments in the later implementation session.

> **Status, 19 October 2026:** The DEC-056 implementation is delivered on branch `dev/recursive-apply-simplified` (reconciliation engine, layout ownership and candidate accounting, simplified execution settings, real Agent turn adapters, bootstrap wiring, evidence, owner channel and guarded publication) and its [runtime contract](design/canonical-model/manager-apply-runtime.md) replaces the process below. It awaits independent complete-diff review, exact-head hosted CI, actual Main composition and owner acceptance; no real-provider run or two-Apply demonstration has been executed. The sequential task machine, deliveries, reviews, reservations, rejection counters, journal and SQLite adapter were deleted. The pre-DEC-056 text below, including the A1 to A6 queue, the zones table and the implementation sequence, is historical; the next dispatches are review and integration, then the demonstration under the owner's [dispatch bounds](concepts/sources/recursive-apply-implementation-20261019.md#demonstration-bounds).

Date: 16 October 2026. This is the delivery and coordination owner, originally commissioned by the [owner](design/canonical-model/sources/manager-realization-20261016.md#polish-the-concept-and-start-the-coordinated-rebuild). It replaces the old waves, role queues, backlog and local dashboard. Old assignments do not carry execution authority into this rebuild.

## Target and baseline

Real Managers realize the small library M1 as A1; independent assessment examines actual code, tests, documentation and history. Change only the model's loan limit from five to three, then Apply M2 against actual A1 to produce and assess A2. Use one frozen product/profile for the final demonstration. No manual code repair between the Applies.

The [canonical language v1](design/canonical-model/minimal-language/language-contract.md) is the language baseline. The [Apply design map](design/canonical-model/README.md) routes technical contracts. The [acceptance gates](design/canonical-model/manager-apply-implementation-plan.md) and [removal map](design/canonical-model/manager-apply-rebuild-plan.md) are specifications, not separate live backlogs.

Starting product: Main `bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310`. Its existing CLI/self-model remain executable until a coherent replacement cutover. The recursive design is implemented only on the branch named in the status note above. Published releases and historical evidence keep their original identity.

## Guiding decisions

The process bullets below record the pre-DEC-056 implementation baseline. Their sequential/private-workspace, per-delivery review and freeze choices are superseded by the current concept. Existing ownership reservations are still relevant to avoiding interference; they are not authority to revive the old process.

- Implement completed v1 and the Manager concept directly. Compatibility is not required; delete obsolete product paths, formats and tests together.
- Current organization remains one ordinary Root and one Manager per local Package across code, tests and docs. [DEC-056](concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace) now selects parallel shared-tree Execute and fresh recursive Verify on one closing stand. The earlier sequential writing and mandatory named per-delivery reviewers/arbiter are superseded for this strategy. Direct parent/child communication and complete participation remain required.
- Separate pure model/snapshot/organization/realization values from execution, repository I/O and provider protocols. Use consumer-owned Ports and Adapters and one Go module.
- Reuse work with current assessment, not old approval. Keep exact bindings, recorded questions and actual effect observations.
- Budget shortage and uncertainty produce visible retained incomplete work. They cannot waive evidence or authorize blind replay.
- Build the technical core first. The four [working-day scenarios](concepts/reflections/working-day-demonstration-20261016.md) remain separate follow-up evidence. Government, experience, doctor, cockpit and extra providers/commands remain deferred. Parallel writing is now part of the DEC-056 direction, not a runtime capability proved by this plan.

## How we work

The owner-adopted [working method](development/working-method.md) governs design learning, a stable complete-workflow target and short owner accounts at milestones ([DEC-038](concepts/register.md#dec-038-learn-through-complete-workflows)). This plan retains delivery scopes, dependencies and integration. The method creates no new backlog, experiment authority or standing approval gate.

### Sessions

The coordinating chat owns scope, dependencies and shared-file reservations. The delivering chat owns its pull request and merge. Separate implementation chats receive one bounded package, isolated worktree, integrated starting base, owned paths, dependencies, acceptance criteria and a short queue of two or three next tasks. They may use independent review subagents. Start at most four implementation writers in the first wave.

Package scopes and dependencies live here. Actual running state, candidate SHAs, reviews and CI results live in each chat/PR; do not create a status file, backlog or dashboard mirror. A chat advances through its authorized queue without waiting for another handoff. A blocked dependency is reported; the next independent queued task can proceed within the same reservation.

The coordinator owns shared policy, this plan except the [architect section](#architect-and-coordinator), root module dependencies, shared-file reservations and every contract and register entry that section does not assign. A bootstrap/cutover package can receive explicit shared-file ownership. Other chats submit a bounded request instead of changing shared files opportunistically.

### Architect and Coordinator

On 17 October 2026 the owner proposed a division of the work between an architect session and the Coordinator, and the Coordinator acknowledged it ([source](concepts/sources/architecture-review-20261017.md#division-of-work)). The owner's words are "Konzept und Herzstück" for the architect and "die Dinge aussenrum" for the Coordinator. The concept stays frozen until the core runs, with the explicit owner-authorized [DEC-043](concepts/register.md#dec-043-project-teams-review-arbitration-and-product-defaults) exception for team configuration and product defaults, and it is completed first ([DEC-040](concepts/register.md#dec-040-concept-freeze-until-the-core-runs)).

Interpretation of that division (marked):
- **Architect session:**
  - the [Apply design map](design/canonical-model/README.md#apply-design);
  - the register entries DEC-024 to DEC-031, DEC-033 to DEC-037, DEC-039, DEC-040 and later entries on the Apply concept;
  - the consumer-owned ports in `internal/run/` and `internal/apply/`;
  - the whole directories `internal/run/`, `internal/realization/`, `internal/organization/` and `internal/apply/`;
  - the `apply` verb wiring, which it requests in the CLI and bootstrap files reserved below.

  These replace the APPLY-ORG, APPLY-REALIZATION, APPLY-RUN and APPLY-INTEGRATE rows of the zones table below.
- **Coordinator and its implementers:** everything else. This includes:
  - the model and snapshot packages;
  - the container runtime, `agentd` and its frame protocol;
  - the session, journal, repository and check adapters;
  - the rest of the CLI and bootstrap;
  - packaging, CI and release tooling, and marketing;
  - the branch and worktree cleanup the Coordinator planned.
- **Shared files** stay reserved explicitly through the Coordinator: `go.mod`, `internal/adapters/cli/command.go`, `internal/bootstrap/main.go`, the model and registry shards outside the architect's packages, the architecture and module gates, and the code map.
- **Port changes** go only through the architect. An adapter that needs one asks the owner, who relays the request.

The owner subsequently instructed the Config chat to integrate DEC-043's concept change on Main so contributors retrieve it there. This documentation-only exception includes its decision/source and exact source registration; it does not transfer Core, adapter or bootstrap implementation ownership. Follow-up implementation must align the versioned config/defaults parser, pure role resolution, instruction capture/rendering, run review roster and arbiter protocol, durable bindings and existing profile producers. Shared-file reservations and actual implementation dispatches remain with their existing owners; no new runtime or study authority follows.

The architect delivers this queue in order, in one or more pull requests each. The Coordinator plans around it. A1 completes the session port contract, in code and in its owning documents, before the rest of the concept. The adapters depend on it, and it adds no core behavior.

| ID | Deliverable | Unblocks |
| --- | --- | --- |
| A1 | Step-bound [session port](design/canonical-model/manager-apply-port-contracts.md#session-and-communication-port). | Codex and Claude session adapters |
| A2 | Core protocol and completed concept: the task machine, messages, issues and composition, with rule IDs and one owner per rule. Open proposals are put to the owner. | A4 and A5 |
| A3 | Producer contracts for `run.Workspaces`, `run.Checks` and `apply.Publisher`. | Repository, check and publication adapters |
| A4 | The core part of P06: one Package under the root through `run` with controlled fake ports. It covers work, independent review, rework, acceptance by replay, the root's integration and the move to Verifying. | A real session through the core |
| A5 | The core parts of P07 to P09: every Manager of the tree, integration at every parent, messages and questions, aggregate verification and reports, then guarded publication through `markitect apply`. | The two-Apply demonstration |
| A6 | Cleanup inside the architect's packages: remove the v1/v2 basis replay and `PassThrough`, give errors a reason, and split preparation out of `internal/apply`. | None |

**The `apply` verb** is composed in the bootstrap on the ports of `run.Config` (see the [runtime contract](design/canonical-model/manager-apply-runtime.md#ports-and-adapters)): the captured settings preparation, the Agent adapters routed by profile, the repository's workspaces, the check runner, the file-based owner channel and evidence, and the guarded publisher. The earlier request for a journal control, `run.Sessions` routing and `ControlledRun` is obsolete.

### Branch names

Use an isolated managed worktree and a non-protected branch from the integrated baseline. New branches use a prefix that describes their scope, such as `dev/`, `fix/` or `docs/`. Never stash, reset unrelated work, force-push reviewed branches or implement on Main. Verify path, branch, full SHA and dirty state before writing.

### Zones

| Package | Exclusive write scope | Dependency and boundary |
| --- | --- | --- |
| APPLY-BOOT | `src/internal/tooling/architecture/`, `docs/development/modules.md`, generated `docs/development/code-map.md`, `.markitect/model/product/foundation-layer.yaml`, the sorted `modelFiles` entry in `.markitect/project.yaml`, generated `docs/markitect/project.md` | First wave. Prepare strict target import/map checks and optional future ownership of pure `internal/model/`, `snapshot/`, `organization/` and `realization/` paths. Keep the required `core-layer` Artifact and existing gates unchanged until cutover; empty future paths need no producer stubs. No producer implementation; root module/CI edits require coordinator agreement. |
| APPLY-MODEL | `internal/model/`, including conformance fixtures/tests | First wave. Own compiler API and complete v1 values. No filesystem/provider calls or legacy imports. Integration waits for BOOT; freeze a reviewed API before dependent packages start. |
| APPLY-SNAPSHOT | `internal/snapshot/`, including tests | First wave. Own fixed generic tree/path/entry/digest values. No Git/process/model dependency. Integration waits for BOOT. |
| APPLY-TESTKIT | Move `src/internal/testkit/` to `internal/testkit/`; mechanical test imports, test-only import/map gates, matching ownership and current guidance | Before repository tests. Share the existing hermetic Git fixtures with root-layout packages without a legacy import or a duplicate helper. Preserve the helper API and behavior; production imports stay forbidden. The coordinator grants these exact shared migration edits. |
| APPLY-PLAY | `experiments/library-playground/` | First wave. Independent Python standard-library oracle and bounded harness checks. No library implementation, Manager engine or real provider invocation. |
| APPLY-ORG | `internal/organization/` | Architect session since 17 October 2026, see [above](#architect-and-coordinator). After MODEL/SNAPSHOT APIs: topology, exact settings, context and mandates from explicit facts. No runtime or registry import. |
| APPLY-REALIZATION | `internal/realization/` | Architect session since 17 October 2026, see [above](#architect-and-coordinator). After MODEL/SNAPSHOT APIs: ownership, census, registry, claims, candidates and composition. No scheduler. |
| APPLY-CUTOVER | CLI/bootstrap except the `apply` verb wiring, self-model, source/coverage/import gates, packaging/smoke, root module and current guidance | After deterministic foundation, read-only Apply preparation and repository capture. One owner switches the active product and deletes old compiler/project/runtime/API consumers. Apply stays explicitly unavailable until implemented. |
| APPLY-RUN | `internal/run/` and consumer-owned Session/Workspace/Check/Journal ports | Architect session since 17 October 2026, see [above](#architect-and-coordinator). After foundation/cutover. One owner for task/review/finding/budget/lifecycle semantics. Establish executable ports before adapters. |
| APPLY-JOURNAL | Retired | Deleted with the journal and its SQLite adapter by DEC-056; there is no journal. |
| APPLY-RUNTIME | `internal/adapters/container/`, `internal/agentd/`, `cmd/markitect-agentd/`, `packaging/runtime/` | Below the session port; may start beside RUN. Container engine client, volumes, limits, the agentd session helper with its MCP bridge and stdio framing, and the pinned base image with both CLIs ([DEC-037](concepts/register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers)). No task, model or review semantics; real smoke needs finite authority. |
| APPLY-AGENT | `internal/adapters/codexappserver/`, `internal/adapters/claudecode/` | Session adapters for Codex App Server and Claude Code on `run.Sessions`, after RUN's Session contract and APPLY-RUNTIME. Profile metadata comes from the runtime image through the engine, which preparation may query but never runs the image; the existing offline metadata preparer is replaced accordingly. Each adapter needs its bounded real handshake smoke; real smoke needs finite authority. |
| APPLY-REPOSITORY | `internal/adapters/gitrepo/`, `internal/adapters/checks/` | Read-only capture follows the reviewed Apply Source/PreviewSource contracts before CUTOVER. A read-only CheckProfilePreparer may follow the Apply metadata port there; workspace creation on runtime volumes, check execution in containers and guarded publication follow the corresponding RUN/Apply contracts and APPLY-RUNTIME. |
| APPLY-INTEGRATE | `internal/apply/` and composed evidence; `apply` wiring through the reserved CLI and bootstrap files | Architect session since 17 October 2026, see [above](#architect-and-coordinator). Before CUTOVER, establish consumer-owned Source/PreviewSource ports and read-only preparation over actual acquired inputs. Model compilation is the first small slice, not a complete check or context. After RUN, add Control/Publisher and orchestration; one Package, recursive integration, verification/publication and real M1/M2 proof follow their adapters. |

A scope is reserved only when dispatched; it is not permission to start every later row. Parent scope grants no sibling files. Shared outputs have one current writer; reservation changes are explicit.

### Work packages

Every dispatch states: package ID, full integrated base, exact paths, contracts, first deliverable, checks, forbidden shared edits and integration dependencies. Producers own exported values; consumers start after the producer's reviewed API slice. No global types library or duplicated ports.

MODEL first supplies an I/O-free compilation entrypoint over explicit source/dependency bytes, typed identities/values/provenance and a result that cannot claim success with partial v1. SNAPSHOT supplies immutable tree/path/mode/content identities and deterministic comparison, distinct from canonical model digests. Exact Go names are producer-owned choices reviewed before consumers start.

New foundation packages may develop in isolation while BOOT is reviewed. They cannot integrate with unclassified paths or disabled import checks. BOOT coordinates ownership and package-map handling without writing producer implementations. The public CLI must not consume the partial foundation before CUTOVER.

Before CUTOVER, read-only preparation and repository capture must support real check/context inputs. Source captures committed clean HEAD with an opaque comparison guard; PreviewSource observes a working draft or an explicitly resolved fixed revision without that guard. Both use the same raw input contract and acquisition mechanism. After the reviewed Apply metadata port exists, Orchestrator adapters may parse source-bound bytes of explicitly selected Check profile metadata using closed formats only; Agent profile metadata comes from the runtime image (APPLY-AGENT). This preparation does not start processes, probe versions, export schemas or invoke providers; passing it does not establish runtime support. Initial context preview exposes the actual model selection, inclusion reasons and resolved settings; full role rendering waits for real task facts. No invented run, task, evidence or budget records make a preview executable. Workspace creation, sessions, check execution and publication remain later work.

### Integration

These rules implement the owner's 17 October 2026 handoff instruction and replace the earlier central merge and per-merge observation procedure.

1. Keep one focused PR per package/API slice. The delivering chat obtains independent complete-diff review and resolves every finding and shared-file dependency.
2. Freeze the approved head and require the configured required CI on exactly that head. A changed head needs renewed relevant review and CI. Wait in the background, through repository auto-merge or a bounded watcher owned by that delivery; do not spend Coordinator turns polling CI.
3. Once review has no open findings and required CI is green, the delivering chat merges its own PR directly with a merge commit and `--match-head-commit`. It needs no second approval or merge action from the Coordinator. Never bypass branch protection.
4. Recompose only when GitHub reports conflicts or the PR overlaps substantively with newly merged work. A newer Main SHA alone does not justify another composition, full test run or CI cycle. When composition changes the candidate, review the relevant delta and require CI on its new exact head.
5. Record the merge result in the delivery chat and move to the next authorized task. Do not watch Main CI or read source artifacts after each merge. React when Main is reported red; investigate the actual failure before assigning a repair.

[CONTRIBUTING](../CONTRIBUTING.md) lists checks. Run local checks relevant to the contract and actual change; required CI supplies the repository-wide gates. Preserve failed observations. Prefer a small distinguishing test for a real error over exhaustive matrices or repeated hardening of unchanged code. Delete obsolete checks with their feature; no skip or allowlist may make an unimplemented replacement look complete. Source integration remains distinct from provider execution, release and owner acceptance.

### Session and process contracts

The real handshake ran on 17 October 2026 ([record](validation/runtime-handshake-20261017.md)). Since [DEC-037](concepts/register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers), every Agent session and project check runs in its own Linux container, and the [rebuild plan](design/canonical-model/manager-apply-rebuild-plan.md#container-runtime) owns the runtime and the session mapping. RUN defines the Session port against that mapping: Orchestrator tools reach the provider over MCP through `markitect-agentd`, and Codex App Server and Claude Code are the two adapters. Codex dynamic tools are not used.

The earlier runtime lessons move into the container runtime:

| Lesson | Treatment |
| --- | --- |
| Terminate the whole process tree on timeout or cancellation ([caeb57c5](https://github.com/Glacius-Labs/Markitect/commit/caeb57c5cca4ff0e27b0f821590a82d5d26dbd94)) | agentd signals the provider's process group, and stopping the container ends every remaining process. Keep a child and grandchild termination test at agentd. No Windows Job Object. |
| Require the exact configured CLI version ([cbe21140](https://github.com/Glacius-Labs/Markitect/commit/cbe211404df4455d7fa26897857757cf38ec1a15)) | The image pins the CLI versions; the profile binds the image ID and the versions it records, and a mismatch refuses the profile. |
| Keep a CLI rejection visible, including for a large prompt ([c1d4ba6c](https://github.com/Glacius-Labs/Markitect/commit/c1d4ba6c30aeeeee7f3489476aa40684d67add56)) | agentd forwards the provider's standard error and exit status, so a refused start reaches the Orchestrator with its diagnostic. |
| Guard Windows short names and aliases ([a726fce4](https://github.com/Glacius-Labs/Markitect/commit/a726fce45e3565513a6f9e13b7a14ec4314af1a8), [837f137e](https://github.com/Glacius-Labs/Markitect/commit/837f137e4f445428a9707aa90d7d9770983f5cd7)) | Sessions never see host paths. The guards stay where the Orchestrator touches the owner's repository: acquisition and guarded publication in `gitrepo`. |

APPLY-RUNTIME builds the container runtime below the session port; APPLY-AGENT builds the two adapters on top of it. Real provider and Apply execution still needs separate finite authority.

## Implementation sequence

1. **Baseline and measurement:** integrate this concept/coordination reset, then dispatch BOOT, MODEL, SNAPSHOT and PLAY separately. The oracle needs no finished CLI; synthetic Apply bookkeeping is not a prerequisite for the pure compiler.
2. **Deterministic foundation:** integrate BOOT, producer API slices and complete v1 conformance; then parallel ORG and REALIZATION. Implement all language constructs, not only M1. Bounded probe evidence remains separate.
3. **Read-only preparation and product cutover:** establish Apply's Source/PreviewSource and metadata-preparation contracts, then model preparation, repository capture and closed-format profile metadata preparation for check/context (Agent profiles later read theirs from the runtime image). This metadata work does not prove runtime support. Switch CLI, self-model and build/coverage/import gates together; remove obsolete source/commands/tests. Keep Main buildable and explicit about unavailable execution.
4. **Execution/adapters:** RUN establishes ports/reducer while RUNTIME builds the container runtime below the session port. AGENT and REPOSITORY then work independently against them, without a second scheduler or ledger (the journal was later retired).
5. **Realization:** one Package through work/review/rework, then root recursion, integration, complete evidence and guarded publication.
6. **Demonstration:** freeze product/profile; real M1/R1 and independent A1 inspection; exact model-only M2 edit; real R2 on A1 and independent A2 inspection. Preserve all attempts.

## Owner decisions

The foundations remain ready for implementation. Before recursive orchestration, align it with [the clarified Manager process](design/canonical-model/manager-apply-concept.md#tasks-and-delegation) and settle its explicitly listed process details. Compatible runtime, storage, candidate and session work can continue; this is not a general rebuild restart. Concrete producer APIs, driver versions and capability identities are bounded implementation deliverables, not reasons to keep expanding the concept. Ask only for genuinely new intent, authority or an exceptional tradeoff.

A pinned runtime image and a bounded real handshake smoke for each session adapter remain mandatory before R1. Credentials and runtime grants must be explicit. This plan does not authorize unlimited provider spending, studies, release/deployment or Government. Existing unrelated PRs/worktrees remain separate unless assigned.

## Releases and historical evidence

The retired backlog/roadmap remain in [Git at previous Main](https://github.com/Glacius-Labs/Markitect/tree/bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310/docs). Old [handoffs](development/session-handoff.md) and [surveys](work-items/surveys/README.md) preserve their scope and cannot restart queues. The local dashboard is retired without a replacement monitor.

The owner retired old release pages and downloads on 17 October 2026. Git tags, captured release identities/attestations, original PRs and needed diagnostics are preserved in the [retirement record](history/distribution-retirement-20261017.md). [Operations](operations.md) owns separate release work; this cleanup changes no Apply-Core contract.

## Historical roadmap links

These anchors route old references to immutable records without reviving queues.

<a id="astra-assessment-disposition-and-next-method-checkpoint"></a>
[Astra disposition](history/implementation-plan-before-product-integration-20261009.md#astra-assessment-disposition-and-next-method-checkpoint).

<a id="standard-operating-model-checkpoint"></a>
[Earlier operating-model checkpoint](history/implementation-plan-before-product-integration-20261009.md#standard-operating-model-checkpoint).

<a id="read-only-policy-failure-analysis-unreleased-source"></a>
[Earlier policy-failure assessment](history/implementation-plan-before-product-integration-20261009.md#read-only-policy-failure-analysis-unreleased-source).

<a id="v0100--canonical-engineering-model-published"></a>
[v0.10.0 historical release](history/implementation-plan-before-product-integration-20261009.md#v0100--canonical-engineering-model-published).

<a id="knowledge-graph-assessment-and-authoring-decision"></a>
[Earlier knowledge-graph assessment](history/implementation-plan-before-product-integration-20261009.md#knowledge-graph-assessment-and-authoring-decision).
