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
| `internal/runtimeconfig` (new) | Closed profile decoding and optional static metadata preparation for CLI/Bootstrap and native/check adapters | Captured Project source values; no Run, provider sessions or credential reads |
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

CLI -> Apply -> Workflow -> Run -> Project/foundations. Diagnostic commands call Project directly. Adapters are injected by Bootstrap. Run cannot import Apply/Workflow/adapters; foundation cannot import execution. Enforce directions for tests too. New package roots require actual consumers and a distinct responsibility. No shared-contracts mega-package, repository framework, DSL, event bus or duplicate model ontology.

Runtimeconfig is the outer preparation consumer for native/check profile metadata. Project and Run do not import it. Its observations retain only the identities, instructions and diagnostics used by those consumers; old strategy, role, helper-team and Turn settings are removed rather than moved into this package.

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

Questions use a small Run-local inbox: Ask returns a correlation ID without waiting for an answer; Await can deliver an incoming question while an earlier answer is pending; Reply carries the addressed Manager's actual answer. A submitted task does not close that Manager's conversation. The host transports questions and answers and validates their association; it does not answer for a Manager.

The native adapters wait for a pending tool's response and cannot supply synchronous Answer-during-Exchange. Reception therefore happens at supported tool/message boundaries, without assuming nested inference or immediate preemption. Direct delegation remains the Manager's responsibility. The bounded Codex hierarchy observation below establishes crossed questions and later answers for its five real Managers. Native Assessment/Verify, complete Apply and each additional provider still require their own evidence.

## Complete Run lifecycle

`run.Executor.Execute` is the single closure owner, including standalone use:

1. Validate checked binding, full topology, mandate and finite allowance. Open one Git Workspace Service for this Run and fresh Manager environments only for its topology. No future-Run reservations. Each Manager retains its copy/conversation within this Run.
2. Install harness and coordinate the full hierarchy. Deliver questions even while other Managers wait; serialize mutations to one parent's copy, not all children. Cancellation releases waiters. Parent reviews/integrates fixed child results in completion order.
3. End active writes before capturing a handoff/Root commit, using ordinary session synchronization and internal Git commits. Bind decisions/checks to that immutable object; a later change requires renewed affected checks/review. No per-write/freeze protocol.
4. Persist required candidate objects and verify independent readability before deleting temporary copies. Retain diagnostic partial candidates where possible; failed retention never yields acceptance.
5. Close every Manager process/container/home, Run Git service and temporary volumes, including partial-start failures. Use bounded cleanup even after work cancellation; no further agent work is granted. Record actual close errors.
6. Save and read back the terminal result including cleanup outcome, then return. Storage failure returns technical error plus available diagnostic identities; never claim a saved result exists. A crash before terminal storage is incomplete, not inferred success or automatic resume.

Cleanup uncertainty, missing decisions and persistence failures block successful return/publication. Standalone caller provides finite limits. Apply supplies one shared concurrency-safe allowance and original deadline to every Run, including corrections. Resource counts have explicit technical meaning, not domain Turn semantics; no reset after Reject.

The bounded native workspace probe at `ac293ffcd5d98f7d3b5710c2f359323dc85280f0` on 23 October 2026 observed both requested `0644` and `0755` files as `0777` through the Windows/Docker Desktop bind. Byte mapping, read-only refusal and cleanup passed; executable-mode fidelity failed. That host-directory binding therefore cannot qualify the native candidate workspace. The failed attempt is retained; successful byte transfer does not waive this boundary.

Use the intended Run-owned Git service with private Linux-volume working copies for native execution. Git operations and each Manager operate on the same copy, with only that Manager's copy mounted into its container. Local Git remains useful for controlled tests; do not synchronize a second writable host mirror. Native/check attachment identifies the actual owned copy without interpreting a host directory as a volume. Export self-contained candidate bundles before closure and verify their commit, tree and file modes through Git objects in an independent empty repository; a Windows checkout is not a filesystem-mode oracle.

The bounded LinuxGit copy proof at source `623eb190b9e0d3d11d29e35bd9c4151e536c3ed6` passed on 23 October 2026 using immutable runtime image `sha256:a3878f3b8aa85b46d829bb5845170052b67b79b8dcece9186dac13b0211efd3f`. In one attempt, actual Linux copies preserved `0644`/`0755`, empty Packages and accounted ignored inputs; sibling copies and disposable check input/home remained separate. Independent bare Git readback after old-resource closure and a fresh Linux service recovered the exact retained candidate. Both Run volumes and all owned containers were absent afterward, with no residue or outer removal. The complete bounded packet took 31.583 seconds and made no provider calls. Its retained record is `C:\Users\Consiliari\Documents\Codex\Markitect-LinuxGit-Copy-Probe\623eb190\attempt-01\package-result.json`, SHA256 `794489c46790f241eeb51d76cc108cf5c5bf8b65b28ff5de6089f2d13ebdca4d`. This establishes that copy boundary on that source, not the later conflict corrections, complete Run coordination, native harness/session behavior or Apply. Those depend on their own reviewed source and subsequent evidence.

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

The implemented Codex path starts the Manager container with agentd awaiting initialization, provisions its private home through one bounded private setup message, then starts the provider. It uses allowlisted paths, rejects symlink traversal, restricts permissions, redacts secrets and keeps credentials outside argv/env/logs/harness/Notes/bundles. Partial setup retains its cleanup responsibility. The tested path installs access-only material from a normal account login; it does not share a mutable host home or copy refresh authority. Expiry, refresh and another provider need their own evidence. No renamed auth-helper container.

Native evidence is an early dependency gate, not a late S4 observation. Prepare the probe during S1 and execute it at the earliest usable S2 transport checkpoint under a concrete bounded coordinator dispatch, before integrating dependent adapters. Without successful evidence, those provider-dependent decisions remain provisional; independent deterministic work can continue. The proof below satisfies this early gate for the stated Codex source and fixture. Claude remains explicitly unavailable.

Remove Codex API-key resolution, profile/config branches, adapter route, examples and branch-specific tests together. Preserve secret validation/redaction tests for account login. Native `turn/start` and Claude conversation identifiers remain adapter facts. Codex and Claude qualification is independent; one does not certify the other.

Notes are persistent project-scoped noncanonical hints with provenance; they do not grant authority or schedule work. Mandatory Findings stay in results and explicit next-Run inputs. Verify sees fixed model/candidate and necessary technical observations, not Implement conversations/self-approval. Notes remain attributed hints. Hidden outside-oracle expectations never enter Implement context.

### Early native evidence

Source `4ebe694a7ed70dd822c4431d704151f5eaeb2514` passed one real Implement hierarchy with Root, two children, a grandchild and an empty Package: Codex 0.162.0, `gpt-6-sol`/high, image `sha256:a3878f3b8aa85b46d829bb5845170052b67b79b8dcece9186dac13b0211efd3f`. Runtime image inputs match `dd7c24d7d7d82935bbff1276a7ec6e4809aa878d`, excluding only the reviewed test and runtime README changes. The bounded dispatch used 18 of 20 Run messages and finished in 199.201 seconds (196.87 seconds in the native test), within 420 seconds of work and 900 seconds overall. Run cleanup was bounded to 60 seconds; separate outer cleanup to 90 seconds.

The independently checked result is Completed/Submitted, with five Sessions, five fixed handoffs, eleven Exchanges and five durable Notes. Recorded interactions establish crossed questions, later answers in the same Sessions, a rejected child followed by repair and accepted integration, and the fixture's canonical instruction precedence. A tracked background writer was observed active, terminated through the native protocol, and stable before handoff. All Session/copy/Git-service closures succeeded; all eight known volumes and mounts were independently absent, with no outer removal, no residue and an unchanged login source. Fresh bare Git readback after closure verified candidate `6f41c293463eb411523cf1ed4ae8b6eb84d08d48`, tree `442276c0e9abe9a8755981d7555374790bea1b26`, and the retained writer bytes.

The retained packet is `Documents/Codex/Markitect-Native-Hierarchy-Probe/4ebe694a/attempt-01` on the execution host. `package-result.json` has SHA-256 `126f9a1cc3b721d163adc5995f6ecd37a24771a5217f99c1430040619c2b462b`; `native-evidence/hierarchy.json` has SHA-256 `071fe0d1769d46d29b2439063ab8dd2c7f80812fd2ddcc7dd214b750118e7bd7`. This is one bounded fixture, not universal instruction reliability, native Assessment/Verify, Apply/publication, detached-writer containment, refresh/expiry or Claude qualification.

Earlier packets remain failures: `27de9f95` stopped at auxiliary recorder exit observation (14.438 seconds); `d5c48b56` passed the loader but supplied an invalid cleanup limit (15.15 seconds); `5c3e9499` reached five Sessions but timed out without handoffs and recorded five Session-close errors plus an initially wrong canonical Note (425.393 seconds). Their original reports remain unchanged with separate diagnoses. The last attempt exposed two concrete boundaries: finish a native response after recording its judgment so the host can drain writers and fix the handoff; and do not begin a new transport write with an already-canceled context before orderly cleanup. Later questions use subsequent messages in the same Session. These corrections add no domain Turn/Pass or alternate completion path.

The unchanged Go runtime recipe at source `2e14fe257deb14589d2432d463d0902be4c5de48` subsequently produced image `sha256:122a9537aba513ef136f9395239d01499bef489d422ed42a83c21d90489e3a49`, retaining all ten qualified base-image layers and the exact agentd/Codex metadata. Docker inspected and ran this OCI index digest directly; it is not the separate image config digest. One bounded build and non-root, read-only-root, network-disabled smoke finished in 41.397 seconds. Go 1.27.1 and gcc were observed; a tiny standard-library test passed with CGO disabled. The owned test container was absent afterward. This establishes component availability, not Library behavior, native Apply, CGO or race support. The retained packet is `Documents/Codex/Markitect-S3-Qualification-Preparation/2e14fe25/go-toolchain-build`; `attempt-01/result.json` has SHA-256 `b00ffe6b9e975a305b5b0fad4f77078a853f3ac7335f1111ec0a073758b9803f`. The dispatch limited the build client to 600 seconds, smoke to 60 seconds and cleanup to 90 seconds within a 900-second observation envelope; it did not claim a hard daemon-build lifetime after client cancellation.

## Acceptance and decisions

First prove a complete small hierarchy using real local Git, controlled sessions, durable result/Notes, cleanup, reopening and publication; then all-mandate freshness and failure branches. Prepare a separately authorized early native probe before broad dependent work. Final qualification covers complete source gates, preservation/no-change, seeded defects and an outside oracle, plus each claimed provider separately.

No unresolved product choice blocks planning: hierarchy, exclusions, Notes and normal login are settled. The corrected plan has explicit source-implementation approval. The subsequent owner instruction delegates useful finite real proofs to the coordinator; claimed release provider support still follows actual evidence. Routine API, package or storage choices do not create extra permission rounds.
