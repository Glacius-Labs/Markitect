# Delivery plan: the first reliable Apply

Date: 16 October 2026. This is the single current delivery and coordination plan, commissioned by the [owner](design/canonical-model/sources/manager-realization-20261016.md#polish-the-concept-and-start-the-coordinated-rebuild). It replaces the old waves, role queues, backlog and local dashboard. Old assignments do not carry execution authority into this rebuild.

## Target and baseline

Real Managers realize the small library M1 as A1; independent assessment examines actual code, tests, documentation and history. Change only the model's loan limit from five to three, then Apply M2 against actual A1 to produce and assess A2. Use one frozen product/profile for the final demonstration. No manual code repair between the Applies.

The [canonical language v1](design/canonical-model/minimal-language/language-contract.md) is the language baseline. The [Apply design map](design/canonical-model/README.md) routes technical contracts. The [acceptance gates](design/canonical-model/manager-apply-implementation-plan.md) and [removal map](design/canonical-model/manager-apply-rebuild-plan.md) are specifications, not separate live backlogs.

Starting product: Main `bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310`. Its existing CLI/self-model remain executable until a coherent replacement cutover. The new design is not already implemented. Published releases and historical evidence keep their original identity.

## Guiding decisions

- Implement completed v1 and the Manager concept directly. Compatibility is not required; delete obsolete product paths, formats and tests together.
- One ordinary root and one Manager per local Package, shared across code, tests and docs. Default teams are Manager plus fresh independent reviewer, with sequential writing.
- Separate pure model/snapshot/organization/realization values from execution, repository I/O and provider protocols. Use consumer-owned Ports and Adapters and one Go module.
- Reuse work with current assessment, not old approval. Keep exact bindings, recorded questions and actual effect observations.
- Budget shortage and uncertainty produce visible retained incomplete work. They cannot waive evidence or authorize blind replay.
- Build the technical core first. The four [working-day scenarios](concepts/reflections/working-day-demonstration-20261016.md) remain separate follow-up evidence. Government, experience, doctor, cockpit, parallel writers and extra providers/commands are deferred.

## How we work

### Sessions

The coordinating chat acts as integrator. Separate implementation chats receive one bounded package, isolated worktree, exact base, owned paths, dependencies and acceptance criteria. They may use subagents inside their assignment. Start at most four implementation writers in the first wave.

Package scopes and dependencies live here. Actual running state, candidate SHAs, reviews and CI results live in each chat/PR; do not create a status file, backlog or dashboard mirror. Integration requires exact-head required CI, independent review, merge and Main read-back.

The coordinator owns shared policy, this plan, the concept register, root module dependencies and cross-package contracts. A bootstrap/cutover package can receive explicit shared-file ownership. Other chats submit a bounded request instead of changing shared files opportunistically.

### Branch names

Use an isolated managed worktree and a non-protected branch from the integrated baseline. New branches use `codex/apply-<scope>` by default. Never stash, reset unrelated work, force-push reviewed branches or implement on Main. Verify path, branch, full SHA and dirty state before writing.

### Zones

| Package | Exclusive write scope | Dependency and boundary |
| --- | --- | --- |
| APPLY-BOOT | `src/internal/tooling/architecture/`, `docs/development/modules.md`, generated `docs/development/code-map.md`, `.markitect/model/product/foundation-layer.yaml`, the sorted `modelFiles` entry in `.markitect/project.yaml`, generated `docs/markitect/project.md` | First wave. Prepare strict target import/map checks and optional future ownership of pure `internal/model/`, `snapshot/`, `organization/` and `realization/` paths. Keep the required `core-layer` Artifact and existing gates unchanged until cutover; empty future paths need no producer stubs. No producer implementation; root module/CI edits require coordinator agreement. |
| APPLY-MODEL | `internal/model/`, including conformance fixtures/tests | First wave. Own compiler API and complete v1 values. No filesystem/provider calls or legacy imports. Integration waits for BOOT; freeze a reviewed API before dependent packages start. |
| APPLY-SNAPSHOT | `internal/snapshot/`, including tests | First wave. Own fixed generic tree/path/entry/digest values. No Git/process/model dependency. Integration waits for BOOT. |
| APPLY-PLAY | `experiments/library-playground/` | First wave. Independent Python standard-library oracle and bounded harness checks. No library implementation, Manager engine or real provider invocation. |
| APPLY-ORG | `internal/organization/` | After MODEL/SNAPSHOT APIs: topology, exact settings, context and mandates from explicit facts. No runtime or registry import. |
| APPLY-REALIZATION | `internal/realization/` | After MODEL/SNAPSHOT APIs: ownership, census, registry, claims, candidates and composition. No scheduler. |
| APPLY-CUTOVER | CLI/bootstrap, self-model, source/coverage/import gates, packaging/smoke, root module and current guidance | After deterministic foundation, read-only Apply preparation and repository capture. One owner switches the active product and deletes old compiler/project/runtime/API consumers. Apply stays explicitly unavailable until implemented. |
| APPLY-RUN | `internal/run/` and consumer-owned Session/Workspace/Check/Journal ports | After foundation/cutover. One owner for task/review/finding/budget/lifecycle semantics. Establish executable ports before adapters. |
| APPLY-JOURNAL | `internal/adapters/journal/` | After RUN/Apply control contracts. SQLite append/control/recovery; coordinator owns dependency pin. |
| APPLY-AGENT | `internal/adapters/codexappserver/`, `internal/adapters/process/` | After Session/process contracts. Pin executable/schema, implement direct stdio; real smoke needs finite authority. |
| APPLY-REPOSITORY | `internal/adapters/gitrepo/`, `internal/adapters/checks/` | Read-only capture follows the reviewed Apply Source/PreviewSource contracts, before CUTOVER. Workspace, check and guarded publication effects follow the corresponding RUN/Apply contracts. |
| APPLY-INTEGRATE | `internal/apply/`, bootstrap/wiring and composed evidence | Before CUTOVER, establish consumer-owned Source/PreviewSource ports and read-only preparation over actual acquired inputs. Model compilation is the first small slice, not a complete check or context. After RUN, add Control/Publisher and orchestration; one Package, recursive integration, verification/publication and real M1/M2 proof follow their adapters. |

A scope is reserved only when dispatched; it is not permission to start every later row. Parent scope grants no sibling files. Shared outputs have one current writer; reservation changes are explicit.

### Work packages

Every dispatch states: package ID, full integrated base, exact paths, contracts, first deliverable, checks, forbidden shared edits and integration dependencies. Producers own exported values; consumers start after the producer's reviewed API slice. No global types library or duplicated ports.

MODEL first supplies an I/O-free compilation entrypoint over explicit source/dependency bytes, typed identities/values/provenance and a result that cannot claim success with partial v1. SNAPSHOT supplies immutable tree/path/mode/content identities and deterministic comparison, distinct from canonical model digests. Exact Go names are producer-owned choices reviewed before consumers start.

New foundation packages may develop in isolation while BOOT is reviewed. They cannot integrate with unclassified paths or disabled import checks. BOOT coordinates ownership and package-map handling without writing producer implementations. The public CLI must not consume the partial foundation before CUTOVER.

Before CUTOVER, read-only preparation and repository capture must support real check/context inputs. Source captures committed clean HEAD with an opaque comparison guard; PreviewSource observes a working draft or an explicitly resolved fixed revision without that guard. Both use the same raw input contract and acquisition mechanism. Initial context preview exposes the actual model selection, inclusion reasons and resolved settings; full role rendering waits for real task facts. No invented run, task, evidence or budget records make a preview executable. This moves only read-only prerequisites forward; workspace, provider and publication effects remain later work.

### Integration

1. One focused PR per package/API slice, with its ID and fixed-base evidence.
2. Independent complete-diff review; resolve findings, shared seams and composition conflicts.
3. Freeze the pushed head while required Linux Quality/Race run. Preserve failed attempts; Windows remains separate diagnostic evidence.
4. Compose with current Main and merge one dependency-ready exact head at a time using merge commits. Never bypass checks or protection.
5. Fetch/read back Main SHA/tree and hosted CI. Downstream dispatch uses the integrated base. Source integration is not release, real-provider proof or owner acceptance.

[CONTRIBUTING](../CONTRIBUTING.md) owns commands. Run focused checks while developing and complete integration gates against the actual target. Delete obsolete checks with their feature; no skip or allowlist may make an unimplemented replacement look complete.

## Implementation sequence

1. **Baseline and measurement:** integrate this concept/coordination reset, then dispatch BOOT, MODEL, SNAPSHOT and PLAY separately. The oracle needs no finished CLI; synthetic Apply bookkeeping is not a prerequisite for the pure compiler.
2. **Deterministic foundation:** integrate BOOT, producer API slices and complete v1 conformance; then parallel ORG and REALIZATION. Implement all language constructs, not only M1. Bounded probe evidence remains separate.
3. **Read-only preparation and product cutover:** establish Apply's Source/PreviewSource consumer contracts and model preparation, then repository capture and complete preparation for check/context. Switch CLI, self-model and build/coverage/import gates together; remove obsolete source/commands/tests. Keep Main buildable and explicit about unavailable execution.
4. **Execution/adapters:** RUN establishes ports/reducer. JOURNAL, AGENT and REPOSITORY then work independently against them, without a second scheduler or ledger.
5. **Realization:** one Package through work/review/rework, then root recursion, integration, complete evidence and guarded publication.
6. **Demonstration:** freeze product/profile; real M1/R1 and independent A1 inspection; exact model-only M2 edit; real R2 on A1 and independent A2 inspection. Preserve all attempts.

## Owner decisions

The architecture is ready for implementation. Concrete producer APIs, driver versions and capability identities are bounded implementation deliverables, not reasons to keep expanding the concept. Ask only for genuinely new intent, authority or an exceptional tradeoff.

A supported Codex executable/schema and bounded real session smoke remain mandatory before R1. Credentials and runtime grants must be explicit. This plan does not authorize unlimited provider spending, studies, release/deployment or Government. Existing unrelated PRs/worktrees remain separate unless assigned.

## Releases and historical evidence

The retired backlog/roadmap remain in [Git at previous Main](https://github.com/Glacius-Labs/Markitect/tree/bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310/docs). Old [handoffs](development/session-handoff.md) and [surveys](work-items/surveys/README.md) preserve their scope and cannot restart queues. The local dashboard is retired without a replacement monitor.

Published assets, original PRs and failed diagnostics retain their identities. [Operations](operations.md) owns separate release work.

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
