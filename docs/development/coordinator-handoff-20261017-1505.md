# Coordinator checkpoint: 17 October 2026, 15:05 Berlin

Preparation draft for the owner's five-hour Coordinator transition. The final stop/checkpoint audit is still pending. This dated handoff does not create a live backlog or renew any closed delivery queue. Read the actual delivery chats and PRs before acting on a later state.

## Resume from current source

The preparation baseline is Main `d60ae93dedb39da112f382a284b33f0ca24c6c08`, freshly fetched after Runtime PR344 and RUN PR346. Fetch Main again before the next assignment. Preserve the [earlier handoff](coordinator-handoff-20261017.md); its two pending deliveries were integrated during this Coordinator run, including the corrected RUN source and its registration.

Read [AGENTS.md](../../AGENTS.md), the [working method](working-method.md), the [delivery plan](../implementation-plan.md), [DEC-037](../concepts/register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers), [DEC-039](../concepts/register.md#dec-039-deterministic-manager-process), and the current [Apply contracts](../design/canonical-model/README.md). Actual entrypoints use `cmd/markitect` and `.markitect/model.yaml`:

```text
go run ./cmd/markitect check --repo .
go run ./cmd/markitect context <owning-package> --repo .
```

Structural preparation, source tests, hosted CI, provider execution, real Apply, releases and owner acceptance remain separate evidence. Public Apply is still unavailable. No actual library A1/A2 delivery or new provider experiment was authorized by this coordination.

## Binding owner instructions

- Every product Agent session and project check uses its own Linux container from a digest-pinned image. Workspaces are Linux volumes; mounts and Host tool admission enforce authority. Host tools use MCP through the single `markitect-agentd` stdio connection. The two adapters are Codex App Server and Claude Code.
- The owner's independent Claude session exclusively owns `internal/adapters/container/`, `internal/agentd/`, `cmd/markitect-agentd/` and `packaging/runtime/`. Do not dispatch another writer, edit or stop that session. No Process adapter, Windows Job Objects or Windows short-name Session workaround.
- New controlled development/review chats use current `gpt-6.1-sol`, high reasoning, isolated managed worktrees, an exact integrated base, bounded paths and short queues. At most four implementation writers. Preserve all worktrees, branches and evidence; do not archive even clean review worktrees without the owner's later archival decision.
- The delivery owns its PR, independent complete-diff review with zero open findings, and background exact-head CI. Both `Quality (ubuntu-24.04)` and `Race detector (ubuntu-24.04)` must pass on the approved head. Self-merge with `--merge --match-head-commit FULL_SHA`, without a protection bypass. Recompose only for conflicts or substantive overlap.
- The owner additionally requires a source Main sync at every Coordinator run: fetch, compare with the last verified Main, inspect new commits/paths and relevant actual contract/source deltas, and steer affected grants or dependencies. Use current integrated Main for new dispatches. Keep preserved historical worktrees and existing exact-head proofs intact; a newer Main alone does not require a rebase or repeated gates. Report meaningful effects, not unchanged snapshots.
- Do not poll Main CI, retrieve source artifacts after merges, revive retired queues/heartbeats, or create a status/dashboard mirror. React to a reported failure. No additional real provider/Apply experiments, releases or deployments without separate finite owner authorization.

## What is integrated

The following fixed heads and merge commits are recorded by the deliveries. Controlled deliveries retain complete independent reviews, exact-head CI and failed observations in their chats/PRs; do not repeat that evidence routinely. The owner's independent Runtime deliveries retain their own bounded proof records and are distinguished below.

| PR | Delivery | Frozen/source head | Merge commit |
| --- | --- | --- | --- |
| [320](https://github.com/Glacius-Labs/Markitect/pull/320) | Corrected original handoff | `270d87c3ec7438fa40b42edc9a47bfefe1a52315` | `412409a22a67c4081fa326b04644b93a61dd10dc` |
| [321](https://github.com/Glacius-Labs/Markitect/pull/321) | Captured Apply input to RUN binding | `b37f14de2630c24cc56f3ff28b1ff8d13629ca5c` | `899538f751287fd70624db99bb15751956a93ad0` |
| [322](https://github.com/Glacius-Labs/Markitect/pull/322) | Corrected bounded Restore and registration | `e440c091a499c8f7bfd42879293d8a4ffae0268e` | `3b3cbdd843315b4b7d0adaec8f66e02f2dfcff15` |
| [323](https://github.com/Glacius-Labs/Markitect/pull/323) | Assignment lifecycle | `d8609789decb69771e08db26e447a5ec2d45f4a9` | `323e3af8ebf0fdec80e2ad8c59d809362f9dd917` |
| [324](https://github.com/Glacius-Labs/Markitect/pull/324) | Actual Apply Control consumer | `990364c51a9d2fc10381a6d7c9676c4618d4a59c` | `4fd20799a20e373b3a0a0930f502168cc08ebcef` |
| [325](https://github.com/Glacius-Labs/Markitect/pull/325) | Provider-neutral Session and durable transport | `724705eadf5ab840d8b7c8474f482b05196397e1` | `3d1a371fbdfdbad8ee77f2a2cee2cd44e951cc18` |
| [328](https://github.com/Glacius-Labs/Markitect/pull/328) | Owner Claude's Runtime source foundation | `a5279241ed3181c700b4efdabda0a0f91507f006` | `d3b209cc0bc810e96b7bafeb76667c5864fc3382` |
| [329](https://github.com/Glacius-Labs/Markitect/pull/329) | Complete recorded inventory, basis v2 | `96801be448fd30471ade2bcefc0b6992d1cdc298` | `94a1ae4d01a74e34f33a832fa17d8c8a2f6fda20` |
| [331](https://github.com/Glacius-Labs/Markitect/pull/331) | Physical SQLite Control/Journal | `99803b599209b335a014fafb49f2d4aa214eb6f7` | `fbffd7fbf67ab1f4939f623268d401e1fbd2c7ba` |
| [332](https://github.com/Glacius-Labs/Markitect/pull/332) | Immutable image metadata observation in Apply | `6beb234b827c09cd9cb64a3ebce2407309e2bcce` | `f8c1acaa169c9955d237e17cd245add963c155bd` |
| [333](https://github.com/Glacius-Labs/Markitect/pull/333) | Historical Candidate/local-census observation | `3e53e6737008513f0be195fd184ecd05db25de44` | `313cd8f8046b3b22cf365d4475cfbb8929289ce1` |
| [334](https://github.com/Glacius-Labs/Markitect/pull/334) | Deterministic Manager process correction | `4744eb6d2fd9d598deb9b9d602abe07b62013fce` | `1daaa4c98e4bc7831397c1685adec680fe5c8e8d` |
| [335](https://github.com/Glacius-Labs/Markitect/pull/335) | Ready without fabricated provider identity | `67cfb65e32572851d90b7757a2dc11d25eeeec90` | `ef3b986676965aa465aaf4bdfd3fd3604e3bae5b` |
| [336](https://github.com/Glacius-Labs/Markitect/pull/336) | Codex image metadata preparer and explicit DI | `c400588f8b0ecb311862e7ba3d24ded05e1902b5` | `f23eaa2072226d905a8d79b8d6c20f419dfdc136` |
| [337](https://github.com/Glacius-Labs/Markitect/pull/337) | Explore barrier/process/rejection-budget contract | `f829982ff61aca93fc6082be01167641d9a5b654` | `c9d69dc51c041bb6ace44fdcb4de88b0c832cff7` |
| [339](https://github.com/Glacius-Labs/Markitect/pull/339) | Real Git/Control/SQLite integration tests | `7962de32e6fc25e6112e568e71108f4e962a03ed` | `363f1c5a8b4116ee1cbb26deec354321714ca9b1` |
| [340](https://github.com/Glacius-Labs/Markitect/pull/340) | Pure pinned Codex 0.162.0 codec | `810cd0c3d7cc3f9a8e3865f218501fd725c0b9c3` | `1bf7e7473d769ac1302db124d03168deac00cdaa` |
| [341](https://github.com/Glacius-Labs/Markitect/pull/341) | Immutable operation and actual Manager renderer | `4876f3b19e66b45d446b9a2c60632436b64e0d4d` | `8db25faff149f8ba0102c95951301e11d4600997` |
| [342](https://github.com/Glacius-Labs/Markitect/pull/342) | Conversation continuity and per-Step target basis | `afb83a946244c43686e6b4369110e7ff3db61166` | `91be3071bd2388cd1ab8f4112e0458cfd9454f10` |
| [343](https://github.com/Glacius-Labs/Markitect/pull/343) | Pure pinned Claude 2.1.301 codec | `620280a53de588523b0a103efadb191e8453e67d` | `20a620b2a77018470cf117c8a9dc1dba920afda6` |
| [344](https://github.com/Glacius-Labs/Markitect/pull/344) | Owner Runtime census/helpers and bounded capability probes | `9d91449466cb35e42b1408db5135ae7ce8591673` | `d60ae93dedb39da112f382a284b33f0ca24c6c08` |
| [345](https://github.com/Glacius-Labs/Markitect/pull/345) | Bounded summaries, idle policy and quiescence contract | `05978bb4faa3ab3b6a7034c53ed996a3554e2659` | `f0ce496373504f47fd5623b25c83fa82b927d013` |
| [346](https://github.com/Glacius-Labs/Markitect/pull/346) | Initial Explore census/gate and live basis v3 | `4f0cdcdea3403fbbf91bf1047a838eecab9a819f` | `aa488be121bf50078cb833268b7c1ffa02100966` |

### Source capabilities and limits

Control now acquires physical identity/locking and restores the original captured basis. Opening it does not append, start an assignment or establish quiescence. The real Git/SQL tests cover revision zero, explicit Prepared acknowledgement, reopen StoreID/epoch/basis, committed drift and state-directory exclusion. Their static profile metadata is explicitly fake.

Journal provides registered repository/store identity, fencing, WAL/FULL/foreign-key configuration, bounded pinned whole-batch reads, exact retries before CAS and verified blob closure through public RUN helpers. Process-kill and directory-sync observations are not power-loss proof. Root dependency work was imported into Journal: SQLite 1.60.1, libc 1.77.1, UUID 1.6.0 and direct x/sys 0.48.0. No further dependency/notices change is granted.

Session readiness is not provider binding or termination. Actual admitted IDs/budgets, permitted redacted pre-send journal bytes and true wire identity are distinct. RUN receives no secret value. Known preparation/Inspect never resends; unresolved effects retain holds. Tool deduplication is channel/call/Step/content-bound; `Fresh` is not tool execution authority and supplies no cross-channel guarantee.

Basis v3 queues every local Manager, ordinary root and empty/intermediate Packages over direct hierarchy edges. The actual compiler/settings/basis and ORG renderer produce a closed inline Explore packet. Scope and MaxContextBytes are bound through the actual Apply producer. Initial Clear requires every attributed confirmed report and recorded stopped/reconciled Session, transport, tool and budget state. Clear does not authorize later work, review, tools or Apply. RO Manager work remains sequential and charged to the reserve-protecting Writing class; it grants no mutation authority or proven parallelism. Required review subjects are N Managers plus P parents, with 2*(N+P) protected starts and Steps. Do not reduce this floor through unproved Session reuse.

V1/V2/pre-readiness histories retain original bytes, floors and receipts. Known retries are found before new live-admission denial. Restore does not recompile or rerender today's model as historical truth. The original pre-policy v2 fixture is 95,017 bytes, SHA256 `db3601a66ba949e6237ef5cc3a20b9c866dd8afadf054ca75a77a1e1469225b4`. CandidateRecord v1 retains actual historical inventory/census and explicitly empty claims; it is not accepted Delivery or successful review evidence.

The Codex and Claude packages are I/O-free codecs, not implemented `run.Sessions` adapters. Codex metadata comes from `runtime.env` of a never-started immutable image via explicit engine DI. Control initialize acknowledgement does not fabricate a Claude SessionID; actual system/init does. Result UUIDs are not Host or provider Step IDs. Raw bounded protocol identities, visible unknowns and strict Unicode/duplicate-key checks do not establish real provider support.

## Accepted target contract still awaiting producers

Read the owning concept/configuration/execution/port documents instead of introducing another specification. Static unchanged Apply reaches every Manager, with initial active read-only Explore and a common barrier. Planning is local without a second global barrier. Host fixes review, acceptance, integration, correction, evidence and publication; AI does not select participation. Communication is direct parent/child only. Parent prerequisites may proceed while children wait.

The owner confirmed persistent Manager conversations, immutable Run input, versioned per-Step working basis plus own unaccepted delta, and immutable review candidates. Basis advance needs exact recorded transition, preserved own work, denied new admission, reconciled provider/Host/late-tool effects, a continuing write barrier and crash reconciliation. A process list or terminal turn alone is insufficient. Changed mounts/authority need a new execution environment; provider resume needs capability evidence, otherwise explicit handoff. Current source's fixed initial binding is a conservative slice, not a promise that every future delivery loses its conversation.

Rejection allowance is cumulative per Run/Manager/closed review purpose: default second rejection triggers parent correction-plan review; third makes affected work incomplete. Success/partial delivery neither counts nor resets it. `rejectionLimit`/`maxRejectionLimit` is the accepted target naming. The rejection counter/engine is not implemented. Current executable YAML still uses actual parser names until a coordinated producer/consumer/example migration.

Bounded versioned child reports must retain mandatory findings and distinguish partial/final/version, with visible split/loading and a total parent-context bound. Idle execution has finite deterministic warm retention/eviction and actual startup cost; waiting is not proof of no effects. Reusing accepted assessments/results and coordination-driven modeling hints are later extensions, not permission to skip the first active Explore.

## Reservations and next controlled proposal

At preparation, every controlled source queue in the table below is closed and its delivery reported a clean pushed checkpoint. Their old task lists are not active grants. The READ-ONLY settings migration proposal is also complete and stopped, with no implementation or shared write grant.

| Role / latest delivery | Chat ID |
| --- | --- |
| Apply binding, Control, image observation / 332 | `01a148ee-c073-7261-a338-133134f9d528` |
| Original RUN lifecycle and Session / 325 | `01a148ee-f8a7-7fe0-888f-e8cbecc5adfc` |
| Recorded inventory and readiness / 335 | `01a14910-3dbb-74c3-9f66-5556963eb1c0` |
| Physical Journal / 331 | `01a14901-102c-7a10-80bf-28a15c810915` |
| Candidate/local-census record / 333 | `01a1492a-c90a-7432-83ae-8c0a5c8fd718` |
| Image metadata adapter / 336 | `01a14923-d502-7ba1-97a5-1b9fc9cd7754` |
| Real Git/Control/SQL tests / 339 | `01a1495f-ec28-7e80-acf8-9ca503dc4e43` |
| Codex codec / 340 | `01a1495f-f152-7d92-a387-9845e9cc3c46` |
| ORG operation/context / 341 | `01a1496d-c74b-7943-a6a6-9bf476d32bb9` |
| RUN initial Explore v3 / 346 | `01a1496d-cbf6-7493-8323-1cf97afae60d` |
| Claude codec / 343 | `01a14981-51df-7430-8963-7318cf99ca87` |
| Completed settings proposal only; no implementation | `01a149ab-f890-7762-94f1-517ed92675f0` |

The completed eight-part proposal identifies the atomic authored parser/defaults/ceilings/public API/consumer/executable-example migration, with exact production/test paths and source-field expectations in its chat. It does not add a rejection engine or role/effect/budget redesign. The clean preserved worktree is `C:\Users\Consiliari\.codex\worktrees\rejection-limit-contract-proposal\Markitect`, branch `dev/rejection-limit-contract-proposal`, pushed and independently read back at `f0ce496373504f47fd5623b25c83fa82b927d013`. Integrated V3 at `aa488be1` was analyzed with read-only Git reads, not checked out or tested there. The wrong initial unqualified context selector failed; corrected owning contexts and structural preparation passed. No tests, implementation, file edits, empty commit or PR were performed.

The smallest proposed RUN production edit is only `internal/run/binding.go`: map modern public settings to unchanged private `managerBinding.ReviewRounds` and `ceilingsBinding.ReviewRounds` wire slots. Do not globally rename historical wire fields or rewrite golden fixtures. Existing versions, strict canonical remarshal, source bytes, context bytes, receipts and review floors stay unchanged. Public Go fields become RejectionLimit and MaxRejectionLimit; authored old/mixed YAML keys are deliberately rejected rather than silently aliased. The producer default remains 3, with actual new source-field provenance; limits, nil/copy semantics and budgets do not change. Source and `configuration.md`'s first executable YAML example must change together. Apply decoding/default/settings files, Organization resolution/rendering, the self execution input, actual consumer test fixtures and a CLI projection assertion are one coherent cut. A future implementation needs a new explicit bounded path grant and actual integrated base; the proposal is not that grant.

The independent owner chat **Consultant**, `01a148ed-f2c8-7f91-85db-76be64df6bfe`, has verified human authority for two-way coordination. Its 334/337/342/345 reservations are released. The owner subsequently commissioned a final concept polish; the new exclusive Docs-only reservation covers `README.md`, `manager-apply-concept.md`, `manager-apply-execution.md`, `manager-apply-port-contracts.md`, `manager-apply-go-architecture.md`, `manager-apply-journal.md`, `manager-apply-library-flow.md`, `manager-apply-rebuild-plan.md`, `manager-apply-implementation-plan.md` and `manager-realization-north-star.md` under `docs/design/canonical-model/`. `configuration.md`, source, model, generated view, root and historical sources are excluded. No new process decision or execution framework is granted. This independent work is pending in the preparation draft; record its final actual delivery or outstanding head/CI/reservation at the checkpoint. Do not stop or archive this owner discussion with controlled chats.

Runtime PR344 has now delivered process census, hardened one-shot `Engine.RunHelper` on volumes, bounded output buffering, frame protocol 2 and `packaging/runtime/examples/go`. Existing Engine/Session APIs include Start/StartProvider, ImageID/ReadImageFile, provider stdout/input, ToolCalls and Stop; see actual source and `cli_smoke_test.go`. The owner's [bounded capability record](../validation/runtime-capabilities-20261017.md) reports real single-run Claude smoke, new-container conversation resume after a workspace change, one-session workspace continuity, census and a Go-image check. These were the independent Runtime owner's probes, not new Coordinator execution. Codex had no runtime credential: MCP readiness was observed, but a real Codex turn/resume remains unproven. No mid-turn interruption, Host-crash recovery or concurrent-session proof is claimed. The census rule is a test assertion, not an implemented Host quiescence policy. Resume of those exact Claude cases is evidence, not a general adapter/Manager/Apply capability. Runtime remains reserved for the owner; route interface needs through them and do not build competing helpers.

## Preserve work and failed observations

The primary checkout remains the owner's `docs/manager-realization-north-star` at `3ebe3c6e6d283b998020350334e24a7303b089b7`. Do not replace it. Coordinator dependency work remains clean/pushed in `C:\Users\Consiliari\.codex\worktrees\apply-coordinator-20261017\Markitect`, branch `dev/apply-journal-dependency-pin`, head `048fdbbb0b43f310c76fc4ec97ed423985d8c6a3`. This handoff uses a different managed worktree, `C:\Users\Consiliari\.codex\worktrees\apply-coordinator-handoff\Markitect`, branch `docs/apply-coordinator-handoff-20261017`.

Keep all delivery worktrees/branches and original evidence. Known recent paths include `manager-explore-v3`, `manager-operation-context`, `git-control-journal-integration`, `codex-session-adapter`, `claude-stream-codec`, `run-session-readiness`, `run-local-review`, `apply-journal` and `realization-candidate-record` under `C:\Users\Consiliari\.codex\worktrees\`, each ending in `Markitect`. Use each actual chat for its complete path/ref; do not infer that a similarly named checkout is disposable.

### Missing original Claude review snapshot

The clean original review worktree `C:\Users\Consiliari\.codex\worktrees\claude-codec-review\Markitect` was archived contrary to preservation instructions. Archived attachment `01a1499b-8d02-7f90-b062-f2f3751879bc` remains. The supported restore failed exactly: `The archived worktree snapshot is missing.` The original directory/environment was not restored.

A separate managed **replacement checkout**, `C:\Users\Consiliari\.codex\worktrees\claude-codec-review-preserved\Markitect`, attachment `01a1499e-a800-73b1-870e-9350e695ab46`, was created through the managed flow. Detached HEAD is the exact reviewed `620280a53de588523b0a103efadb191e8453e67d`, clean, diff to that commit zero, all eleven delivered files present. This preserves a source checkout, not the missing original snapshot. The implementation worktree/branches, original review transcript and CI remain. [PR343 preservation note](https://github.com/Glacius-Labs/Markitect/pull/343#issuecomment-6233374221) records the distinction. Keep both attachments; do not repeat cleanup or claim full restoration.

### Closed failures remain evidence

- The dependency pin's original Windows-only check missed a Linux UUID checksum. The full reviewed branch includes the correction and actual readonly CGO0 Linux/Windows graph/build verification; Journal imported both commits, not the incomplete candidate alone.
- Journal's first review found directory-fsync and pre-open sidecar/continuity defects, corrected and independently closed. Its first hosted Race attempt hit the bounded pagination deadline, not a DATA RACE. Smaller ordinary pages retained the operation cap and maximum whole-batch ceiling; final exact-head Quality/Race passed.
- CandidateRecord and RUN retained unrelated Windows agentd Unix-socket failures as failed local observations. Protected Runtime was not patched or exempted; final required hosted Linux gates passed. RUN also corrected its actual Bootstrap budget/import test failures and retained their logs.
- RUN's V3 stored-setting source P2 was independently closed with actual inline-byte/whole-document validation and history-negative probes rebuilding valid outer hashes. Stopping a Session no longer hides unresolved transport/tool effects. Unsupported phases' old budget/session arithmetic moved to explicitly internal pure units; no public permit hook or synthetic new legacy history was added.
- PR342's first executable YAML example used target names before the parser did and failed Quality. Its corrected final head kept executable current names and accepted target prose; full renewed review/CI passed. Do not reuse the old head's approval for a changed head.

## Final checkpoint procedure

At or before 15:05 Europe/Berlin on 17 October, stop new controlled slices. Collect the settings proposal and all outstanding controlled checkpoint acknowledgements. Preserve exact pending review/CI if any; do not trade quality for the deadline. Complete independent review and relevant document/accounting checks for this handoff, commit/push and record its PR/head. A pending gate remains pending explicitly.

Disable this Coordinator's follow-up `markitect-apply-koordination-bis-zur-bergabe` and its one-time checkpoint `markitect-coordinator-handoff-nach-f-nf-stunden` after the handoff is published. Never reactivate the old `markitect-apply-weiter-orchestrieren`. The current root chat is `01a148ea-b13c-7342-8ea8-7da28d95740f`. The owner can then archive controlled chats and start a fresh Coordinator using this handoff and the actual latest delivery reports. Independent Claude Runtime and Consultant are not stopped by that transition.
