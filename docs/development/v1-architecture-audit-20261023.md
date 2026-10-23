# V1 architecture audit and v0.17 counter-review, 23 October 2026

**Subsequent owner decision:** the counter-review below retains its planning-time evidence. The owner has since [accepted the corrected plan and authorized source implementation](../concepts/sources/v1-source-implementation-approval-20261023.md); historical hold statements below no longer block that source work. The subsequent standing evidence authorization in that same record delegates useful bounded real proofs to the coordinator; it does not change the historical findings or their evidence limits.

Result: support the narrow new execution core, with the concrete corrections below. This consolidates the prior audit rather than opening another plan series. The [latest owner request](../concepts/sources/v1-scope-clarification-20261023.md) is binding; the supplied v0.17 design is a proposal under review. Earlier v0.16 preservation requirements and this coordinator's earlier Root-only/consumer-migration recommendations are superseded. No implementation is authorized by this report.

## Source and authority

Local HEAD/Main and remote Main checked with `git ls-remote origin refs/heads/main` are all `447ac3b152de79b52de80513e14dc94349a060bd`. Primary checkout is clean; planning worktree `v1-architecture` is on `docs/v1-architecture-plan`, same base, with only this session's documentation/control-input drafts. Thus no new Main changes need separation from the supplied code review.

The commit is a root source snapshot, message `Preserve Markitect source snapshot; reset history`, with no parents. Historical objects `4cbfcb005d9a3d0636139314955f070d3abdb219`, `c1ead43d` and `6e620d7dfeabccfedfe265f6010714afecd3bb42` are locally readable as commit objects; neither ancestry nor remote retention follows. Before later removal, preserve a verified durable snapshot, not merely a dangling local object.

Inputs read in full:

- User request, attachment `b93d836e-897e-43e8-996d-66abadafa5a9/Pasted text.txt`, SHA256 `0AE9103FCE3A3F59538828E3D2EE8D08FA8225B48C1C9F3A47E407D8F131F474`; archived at the owner-source link above.
- Downloads `coordinator-gegenpruefung-prompt.md`: explanatory parallel assignment; the pasted user request controls scope.
- Downloads `markitect-v1-plan-v0.17.md`, SHA256 `0F651AB664A54970F8274AA09D54016EBDC971629D73CEA264EF45F72EEC1595`: proposed architecture, not automatic authority for every technical detail.
- Downloads `code-review-20261023.md`, SHA256 `8AC3617D02464A8FEA73E8F70715BC90750BF2E42C5CBF88D77D937F12E2BC9E`: targeted static review at the same Main, explicitly without its own Go/runtime test claims.

## Confirmed source findings

| Finding | Source and consequence |
| --- | --- |
| Public Apply is already assignment-based | `bootstrap/execution.go` Execute/realizeAndPublish/withPersistentGit calls RealizeAssignments and PublishAssignment inside an Apply-wide Git lifetime. Do not call the public entry an old Pass engine |
| Reviews are separate single assignments | `apply/assignment_realization.go:116,124,152` surrounds OpenAssignmentTree with ReviewAssignment; `run/assignment_review.go` owns focused review. Full hierarchy in all three mandates requires actual replacement, not a wrapper |
| Verify corrections retain Implement | `assignment_realization.go:178` calls `result.Work.Correct`; `run/assignment_schedule.go` reuses its scheduler/sessions. New correction must be a fresh Run |
| Results/publication carry live objects | `assignment_realization.go:58` includes StartingWorkspace and Work; `assignment_publication.go:13` requires result.Work. New durable result and publication cannot retain that dependency |
| Future review volumes are reserved | `apply/assignment_workspace_ids.go` reserves up to maxMessages Verify copies; `gitrepo/git_session.go` StartGit uses instance maps and CloseGit leaves volumes. Volume survival is not independently reopenable candidate storage |
| Domain/native Turns differ | `run/agents.go`, `turn.go` retain old domain API. Codex OpenAssignment internally builds run.Turn and runs native turn/start. Keep native protocol facts; no domain Turn/Pass inheritance |
| Auth helper/API-key routes exist | `codexappserver/agents_credential.go`, `agents.go`, `profile.go`, `bootstrap/runtime_credentials.go`: API-key branch plus helper issue/remove. Safe same-container preparation is not an existing session capability |
| Full organization already has suitable pure semantics | `organization/topology.go` BuildTopology includes empty/intermediate local Packages; content digest alone does not bind that inventory. Preserve semantics, not runtime configuration |
| Model/accounting foundation remains useful | `model/check.go`, `snapshot/tree.go`, `realization/accounting.go`: deterministic checked values and exact classification. Accounting does not receive acting Manager/task base |
| Shared build couples excluded consumers | `bootstrap/main.go` composes Extract/Setup/Cockpit; `authoring/session.go` uses old Run contracts; CLI, modeldocs, checks, providers and Git share Apply/Run types. Removing only verbs does not remove Go package coupling |
| Model blocker already stops correction | RealizeAssignments recognizes ModelProblem after work and Verify. Preserve this behavior; do not invent a new bug claim |
| Existing architecture gate is red | `run/assignment_schedule_test.go:15` imports concrete Codex adapter. Preserve the frame/codec regression in adapter tests and provider-neutral Run tests; no gate exception |

Useful behavioral expectations include exact task/result commits, reviewed serialized parent integration, cancellation/queue release, shared allowance, independent exact candidate approval and guarded publication. Existing code is evidence for those expectations, not a compulsory reuse unit. The deterministic core remains protected; runtime categories are replace/remove or selectively reuse only after suitability inspection.

## Corrections to the supplied plan and earlier drafts

1. Full hierarchy is a settled requirement, not a mandate-dependent optimization. Remove this coordinator's previous Root-only-when-sufficient exception and focused review special path.
2. Remove Extract/authoring/UI migration and Extract-specific contract requirements. The new retirement inventory covers actual source/test/import consumers. Proposed setup retirement includes explicit tested configuration examples. Retain modeldocs/context as model-maintenance diagnostics, not UI scope.
3. Replace interface-first parallel waves with a small complete standalone Run and an immediately following complete Apply/publication proof. Publication is too central to postpone behind a late product-completion phase.
4. Name `run.Executor.Execute` as the complete lifecycle owner: candidate retention -> all resource closure -> terminal record save/readback -> return, including standalone use. Workflow never completes storage on Run's behalf. Failed result persistence returns failure, not an invented record path.
5. Use a single Harness/Provider owner from context to actual native loading. Explore the missing agentd private setup capability early. Account-login expiry/refresh, native instruction priority and secure cleanup need provider evidence; do not freeze interfaces assuming they already work.
6. Keep self-contained Git bundles as a concrete simple proposal; verify objects from an empty repository after old resources disappear. Publication is independent host Git over stored candidates/results. Apply-wide retained volumes are not the candidate contract.
7. Ownership checking uses real task diffs plus accepted child commits/integration bases. Do not infer author identity from registry accounting or subtract all child-owned paths from parent changes. Overlapping edits/conflict resolution require review in context, not a new global provenance system.
8. Make integration honest: excluded consumers can retire first, but the mixed Run/adapter contracts need an atomic buildable replacement boundary. Never merge a broken graph or hidden second engine. A narrow intermediate CLI may explicitly withhold Apply until new composition lands; it must not fall back silently.
9. Do not turn logical responsibilities into packages/interfaces by counting them. Proposed Project, Harness, Workflow and Runstore boundaries each have actual consumers and enforce dependency direction. Candidate Git storage stays beside Git; result and Notes storage share one concrete adapter. No general recovery/workflow/persistence framework.
10. Default publication to conflict on relevant user-target drift. A guarded transfer does not guarantee atomicity against an uncooperative concurrent writer; partial transfer/readback must be observable. New semantic content cannot inherit an old Verify approval.

The [architecture](../design/v1-execution-architecture.md) owns the concrete contract/package decisions; [assignments](v1-workstream-assignments.md) own the bounded retirement/source scopes. No product choice needs reopening. Explicit plan approval, later finite native execution authority and evidence-backed release support matrix remain owner decisions; ordinary technical detail stays with implementation.

## Validation actually performed

During this v0.17 counter-review, production source remained unchanged:

- Current Main/remote identity, worktree/dirty state, source-reset parents and local historical-object availability checked.
- `go run ./cmd/markitect check --repo .` then `context product --repo .` passed in the isolated planning worktree before consolidation.
- `go test ./internal/model ./internal/snapshot ./internal/organization ./internal/realization -count=1 -timeout=3m` passed, all four packages, on unchanged Main source.
- `go run ./cmd/markitect-check-architecture --repo .` failed, exit 1, with exactly the unchanged Run-test -> Codex-adapter dependency above.
- Read-only cutover reviewer ran `go list ./...` successfully; it did not run tests, providers or containers. Lifecycle reviewer performed static analysis only.

Previous audit in this same task had additionally passed focused assignment/Git/bootstrap tests with selector `^(TestAssignment|TestFocusedReview|TestApplyGit|TestIndependentAssignment|TestControlledRootLibrary|TestChildGit)` in run/apply/gitrepo/bootstrap. Those tests were not rerun for this static counter-review and do not qualify the proposed replacement. The supplied independent review reports static evidence only; its claims were not converted into self-run tests here.

No full Go suite/build, Linux/race, hosted CI, container, real provider, product Apply, release or deployment ran in this counter-review. Source tests do not prove native harness/auth, cross-Run candidate retention or target semantic reliability. Earlier [Library Run 08](../validation/apply-assignments-luna-20261023.md) retains its own historical source and failures; it is not v0.17 qualification, M1 -> M2 preservation evidence or successful Claude support.

## Independent review and closure

Fresh read-only analyses checked source retirement and runtime/lifecycle/auth boundaries. A separate fresh reviewer inspected the consolidated tracked and untracked planning documents, then reviewed the correction delta. It found five issues: ambiguous neutral/native harness ownership, insufficiently explicit early native dependency gate, S1 shared-preparation/build transition, session-port ownership and historical entrypoint wording.

All five are resolved. Native renderers belong to provider adapters; S1 retains shared non-Extract preparation until S2 atomically rebinds all retained consumers; S2 owns the session port and the provider owner its implementation. The S1-prepared probe runs at the earliest usable S2 native checkpoint only with separate authority; provider-dependent contract freeze/integration stays blocked without evidence. Old entrypoint paragraphs are marked source history. The independent delta review reported no remaining material contradictions in those boundaries, full hierarchy, Run closure/persistence/cleanup, child integration or closed-Run publication. This is design review, not implementation qualification.

Final validation: documentation links passed for 29 default documents plus nine explicit changed pages; structural `check` passed after source registration; tracked and all five untracked documents passed whitespace checks. The archived owner request has the exact SHA256 recorded above. `git diff --exit-code HEAD -- internal cmd .markitect/model .markitect/model.yaml .markitect/execution.yaml go.mod go.sum` passed: production code, canonical self-model and runtime inputs are unchanged. Primary Main remains clean. Only local planning documents, authority pointers, exact documentation registration and archival text attributes changed. No commit, PR, merge or implementation chat was created. Implementation remains held.
