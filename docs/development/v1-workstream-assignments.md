# V1 bounded implementation assignments

Corrected v0.17 assignments, accepted for coordinated source implementation in the [owner approval](../concepts/sources/v1-source-implementation-approval-20261023.md). These assignments replace the old F/H/G/E/N/P/L migration sequence. Read the [current plan](../implementation-plan.md), [architecture](../design/v1-execution-architecture.md), [counter-review](v1-architecture-audit-20261023.md) and [latest owner instruction](../concepts/sources/v1-scope-clarification-20261023.md). Source changes, bounded provider/container-free tests, reviews, PRs and integration are authorized; real container/provider/product Apply execution still needs a separate finite grant. Use Sol High and at most two concurrent writing chats.

## Common dispatch and ownership

Before each later dispatch record exact integrated SHA, branch/worktree, dirty state, writer identity, exclusive paths and required checks. Planning base is `447ac3b152de79b52de80513e14dc94349a060bd`, not a permanent future base. One writer per shared contract, registry shard and generated file; consumers start from reviewed integrated signatures. At most two concurrent writers initially. Independent review is read-only and covers the complete integrated diff, including deleted tests, source inclusion and self-model. Freeze reviewed head, merge under policy, read Main back.

Source changes include exact artifact accounting, architecture imports/code map and relevant guide updates. The slice's integration owner alone edits shared `internal/tooling/architecture/{target.go,layers.go,imports.go,codemap.yaml}`, source/release inclusion, `docs/development/code-map.md`, `.markitect/registry/**`, `.markitect/model/**` and generated model views as needed. Do not reserve every file to multiple streams. Actual dispatch narrows this list to observed files and transfers it explicitly.

## S1: foundation and excluded command retirement

Owner: Foundation/Command. Characterize the Project preparation seam and keep shared non-Extract preparation in its current package until S2. Preserve model language/semantics and positive/negative regression expectations. Keep `internal/modeldocs` on current preparation during S1; retain `check`, `context`, `docs`, help/version/licenses and future `apply`. Check requires no provider/container metadata. Retire setup and provide tested explicit runtime/run configuration examples; do not create a new onboarding project.

Proposed retirement inventory, all relative to the repository:

| Active scope | Planned action |
| --- | --- |
| `internal/adapters/cockpit/**`, `internal/authoring/**` | Remove excluded Go packages, embedded UI and package tests from the active source tree; preserve prior source snapshot |
| `internal/adapters/cli/{extract,cockpit,setup}.go` and corresponding tests | Remove verbs/options/help/report surfaces; edit `command.go` dispatch and `execution.go` shared Apply/Extract branch to Apply-only |
| `internal/bootstrap/cockpit*.go`, `setup*.go` and related tests | Remove excluded composition; remove `WithExtract/WithSetup/WithCockpit/WithWindowOpener` wiring and Cockpit import from `main.go`; remove Extract method/report route from `execution.go` |
| `internal/apply/extract*.go` and tests | Remove excluded workflow/preparation/accounting/brief/texts; no porting to new Run |
| `internal/apply/model.go`, `preparation_capture.go`, `preparation_settings.go`, `file_check.go`, `file_check_run.go`, `assess.go`, `proposal.go` | Remove only `CapturedModel.extract`, capture/settings Extract hooks, `CaptureExtract`, artifact-suppression branches and Extract assessment/proposal bindings. Retain shared functions/files and ordinary preparation used by current Apply/diagnostics until S2 |
| `internal/apply` deterministic model/source/registry/context preparation, `internal/modeldocs` | S1 retains shared non-Extract preparation and modeldocs callers unchanged in package placement. S2 moves/rebinds all retained callers to Project atomically; no temporary compatibility bridge or Run import in Project |
| CLI/config examples, `docs/project-workflow.md`, README/help, source packaging, smoke expectations | Stop advertising excluded functions; explicit invocable V1 config path replaces setup |
| Model/registry, architecture maps/import tests, generated code map/model view | Remove exact retired artifacts, update self-model support scope and package ownership; do not change model language |

The inventory names source families and shared hooks actually found, not permission to delete arbitrary matching files. Dispatch enumerates exact current tracked files and shared consumers. No broad build tags, test skips, dead Go copy in another active folder, hidden legacy binary or unavailable stub advertised as supported functionality. Historical evidence docs can remain clearly historical.

Before any deletion verify the [durable source archive and restoration instructions](v1-source-archive-20261023.md). Named remote ref and independent full bundle already retain the exact pre-cutover commit; its checkout/fsck/commit/tree readback passed. Do not delete the archive during implementation cleanup.

Validation: focused foundation and negative diagnostics, CLI command/help tests, modeldocs guarded write tests, normal `go build ./...` and relevant suites, structural accounting, precise architecture map. Existing architecture failure is repaired by moving the actual codec/frame regression to adapter tests and preserving provider-neutral Run coverage; do not add exceptions.

## S2: complete standalone Run and atomic replacement

Owner: Project/Run/Git/state integration. Depends on S1 retirement and foundation characterization. In this slice move deterministic preparation to Project and rebind all retained check/context/modeldocs/new-Apply callers together; remove old shared preparation only when those callers compile. Scope also replaces `internal/run`, affected Git/check ports, concrete Git retention and `internal/adapters/runstore`. This owner owns the consumer-local Sessions/session signatures; Harness/Provider owns their native implementations. Provider-dependent contract details stay provisional until the early native capability gate succeeds. Include affected Bootstrap/CLI wiring only after Foundation owner releases it. Shared registry/gate files have this owner during integration.

Deliver a standalone Execute through Root + two children + grandchild + empty Package. Real local Git, controlled sessions and a real file store must cover delegation/questions, parallel work, reviewed child correction, integration, fixed Root candidate, Notes, candidate export, cleanup, terminal save and reopening from a new instance. Run owns closure, not a caller's deferred Cleanup. Include failed start, cancellation, crossed questions, budget, candidate/result storage failure and uncertain cleanup.

This integration cut also removes legacy `Agents.Run(Turn)`, Pass/Turn execution, review/probe/strategy/progress machinery and the separate `ReviewAssignment` engine. New native adapter/check/Git consumers must compile against the new small ports before removing old types. Replace former `assignment_*` code only where its implementation suits the new architecture; no preservation commitment. Keep behavioral regression expectations for exact handoffs, integration conflicts, shared allowance and truthful stop observations.

Do not split this into a merged interface-only core and broken adapters. Bounded commits can be reviewed continuously in an isolated branch, but integrate only one buildable executor path. Native implementation not yet proven may return an explicit unsupported/not-ready result; it must not forward to the old runtime. Until S3 lands, public Apply clearly reports that the new command composition is not yet available. Test doubles are test-only, never a shipped simulated success path.

## S3: complete Apply and publication

Owner: Command/Publication, after integrated S2 and explicit handoff of Apply/CLI/Bootstrap. Add small `internal/workflow` and replace `internal/apply` application. Consume Project, Run and stored results; do not introduce another candidate/result store.

Complete Assessment -> Implement -> Verify with full topology every time. Verify Reject passes required Findings into a fresh Implement, then a fresh Verify. Confirmed model blocker at any mandate ends without publication. All Runs share original command deadline and allowance. Use the four checkpoints and exact accepted candidate. Root/child errors are repairable feedback; successful Reject of invalid candidate remains distinct from technical failure.

Publication loads stored acceptance/candidate after every preceding Run container/service closes, checks clean target/source guards, transfers without user commit, and reads back. Initial implementation fails on relevant drift, rather than merging new unverified contents. Test drift, partial writes, readback failure and preservation of concurrent user changes. Narrow CLI reports actual terminal result and persistence/cleanup errors consistently in stdout/JSON; no UI migration.

The first command vertical proof is here, before broad native qualification. It does not bypass the earlier native capability gate for provider-dependent decisions. Its fixtures use real Git and controlled sessions; they do not claim real provider or Docker readiness.

## S4: complete harness and native providers

Owner: Harness/Provider, one owner across `internal/harness`, Codex/Claude adapters and reserved container/agentd setup files. Preparation/feasibility work starts alongside S1, consuming model/context examples and investigating actual native instruction priority, message transport and safe account-login provisioning. S2 alone owns the Run consumer-local session signatures; this owner supplies native implementations and evidence through one reviewed handoff. No competing core contracts.

Own all five harness layers through actual native use. Test version/input determinism, context completeness, instruction priority, available tools, absence of unintended repository writes, Notes access and full modeled hierarchy. Codex/Claude protocol facts may be reused; old Run/Turn API cannot.

Auth changes cover `codexappserver/{agents.go,agents_credential.go,profile.go}`, `bootstrap/runtime_credentials.go`, associated tests, runtime/profile examples and provider/project guides. Remove API-key input route, validation/default/example branches and auth-issue/auth-remove helper containers. Preserve account-login secret validation, redaction, permissions, expiry and failure tests. Coordinate any currently setup-owned credential path with S1 retirement. Claude receives the same lifetime/harness guarantees with native-specific authentication handling.

Preferred technical spike: narrow private agentd initialization in the already started Manager container, before provider startup. Prove fixed safe file paths, permissions and cleanup without logging secrets or mounting mutable host credential homes. This capability is not present today. Controlled transport tests can be developed after source approval; a real login/container/provider test requires a separate finite grant. Report a mechanism blocker promptly; do not bury it behind a nominally complete pure builder.

Prepare the smallest finite real probe and approval packet during S1. At the earliest usable S2 transport checkpoint, before provider-dependent session/setup signatures freeze or dependent adapters integrate, execute it only under a separate grant: pinned product/image/profile, one small complete hierarchy, explicit messages/deadline/cost ceiling where enforceable, auth/harness load and actual cleanup observations. Test Codex and Claude independently. No automatic retries until green. Missing authority or failed capability evidence blocks provider-dependent contract freeze and native adapter integration. Independent provider-neutral deterministic work may continue, but does not waive the gate. Do not wait for a full S3 Apply to test this narrow native setup/session path; later full native workflow qualification remains S4/S5.

## S5: cleaned final qualification

Owner: integration/qualification, independent architecture reviewer. Ensure one executor and one direct Apply workflow, no live result dependencies, no orphaned legacy contracts/imports/strategies/config/help/fixtures. Review full import graph and actual call sites, not just package counts. Final self-model, registry and generated readable view describe the supported narrow product; historical docs are routed out of current entrypoints.

Run appropriate full source suites, build/static/architecture gates, Linux/race and explicitly dispatched hosted checks on the frozen cleaned candidate. Record unavailable checks honestly. Real container/provider qualification remains separately bounded. Required scenarios: full hierarchy in all mandates; crossed questions and correction; model blocker; four checkpoints; independent candidate reconstruction; standalone terminal persistence; harness actually loaded; cross-command concurrent Notes; fresh Library, loan limit 5 -> 3 and no-change; seeded integration defect/outside oracle; correct child integration versus out-of-area parent changes; exact approval/publication, drift/partial failure; timeout/start/storage/cleanup failure.

Name each provider's supported matrix and the evidence/repetition policy before final real execution. Preserve failed attempts. Deterministic tests, native runtime, hosted CI, source integration, release and owner acceptance are separate outcomes. No release/deployment work is commissioned by this plan.
