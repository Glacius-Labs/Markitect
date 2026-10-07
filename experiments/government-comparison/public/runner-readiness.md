# Runner and common adapter readiness

This is a finite readiness package, not a comparative result. Live trials: **0**.
Actual provider/Actor probe sessions: **0 of the authorized maximum 2**. No model
was substituted and no new authentication, purchase or user-config change occurred.
The common public adapter contract remains v1.1.

## Decision and enforced boundary

The coordinator selected the existing `commonLimits` in `resource-proposal.json`
unchanged: per task 1200 seconds, 12 Actor sessions, 80 provider calls and 120000
tokens; per trial 7200 seconds, 72 sessions, 480 calls and 720000 tokens; parallel
4, transport retries at most 1, semantic repairs at most 2, active human time 600
seconds. All setup, model creation/upkeep, ministries, descendants and reviews
consume the same profile. Selected limits are not a claim of technical enforcement.

The separate tiny runner authorization permits at most two fresh sessions, each
180 seconds and six provider calls, parallel 1, no descendants or retries, and a
10000-token aggregate stop threshold as counters arrive. The installed runner's
CLI help and generated app-server schema did not establish a pre-dispatch provider
call limit. `exec` reports `turn.completed` for an agent turn; that event is not
evidence of the number of provider requests inside it. Consequently the package
does not launch a provider probe. Account support and resolved model identity for
the requested `gpt-6.1-sol`, reasoning `high`, remain a concrete readiness gap.

`runtime/adapter.py` actually handles the v1.1 `--request`/`--result` interchange,
binds the SHA-256 of exact request bytes and preserves raw native stdout, stderr,
command/cwd/exit/time receipts. It calls the pinned Classic release's real
`version` command or the pinned Codex executable's real `--version`. Government
returns `readiness_gap` until a native handoff and G1-G5 receipts arrive. Study
tasks, live mode and persistent resume remain closed; none is simulated.

`runtime/process.py` launches only after assigning a waiting launcher to a Windows
Job with kill-on-close. A deadline, STOP sentinel, log-size stop or parent closure
terminates the owned process tree. A failed Job assignment never releases the
target executable. This is process lifetime control, not filesystem isolation.
Native readiness probes have a 30-second cap and zero Actor calls. The deadline
poll interval is 25 ms; OS scheduling and termination can add latency. POSIX
process groups are a portable implementation path, unverified in this checkpoint.

`runtime/ledger.py` uses SQLite `BEGIN IMMEDIATE` transactions to reserve sessions
before launch, keep setup/reviews/failed starts in the same trial, reject changed
identities or budget refills, enforce session/parallel limits on admission and
retain stop/human intervention state. Unknown completed usage blocks the next
Actor; it never becomes zero. Known retrospective usage blocks further admission
at the selected limit but cannot prevent an in-flight overshoot. Concurrent actors
do not have provider-token/turn reservations. Trial/task elapsed time uses the
persisted system clock; process deadlines use a monotonic clock. Active calls and
crashed reservations are not automatically forgiven. Transport/repair limits have
no live dispatcher yet. The ledger is tested infrastructure, not a demonstrated
end-to-end study budget controller. No Actor dispatch path is enabled by this package.

## Configuration, context and provenance

The actual native runner is `codex-cli 0.130.0`, SHA-256
`280cb1c4e3375d94dbdcba1a191f4f6adbf73c293be1e4f16c74b006662b9c54`.
`runtime/runner.py` records its exact executable and a prospective common profile,
with `--ignore-user-config`, `--ignore-rules`, ephemeral sessions, explicit model
and reasoning, a zero project-AGENTS byte limit, disabled memory, plugins, hooks,
goals and subagents, and zero provider transport retries. These settings are
proposed explicit inputs; their effective application, including global or
managed instructions, has not been validated in an Actor session.

An earlier native `login status` inspection failed while parsing ambient config
at `C:\Users\Consiliari\.codex\config.toml:7:1`: string `gpt-6-luna` where
`AgentRoleToml` was expected. The file was not edited. The intended `exec`
`--ignore-user-config` path is documented, but auth/model execution was not exercised.
No credential values are included in the package.

The native version probes use new independent empty Git roots outside every
Markitect checkout under a separately named Documents directory. Their only
released input is a newly created public sentinel. This is preparation, not an
Actor access test. No study source, holdout, future cards or cell results are sent
to any Actor. Cooperative separation and equal prospective context policy are
documented; actual filesystem read/write/process isolation and effective context
must still be measured. A read-only/workspace-write flag alone proves no privacy.

Classic is the published v0.14.1 binary from source
`7dbd599c81540c8203a1b7f83afbc335174f4f1f`, SHA-256
`2cad55efad64f15d7f57638bea78312918fbf7d181c730f188b9921504da71c4`.
The held `c91363b7ac4decbe87212ff0f588b5451581a152` is a distinct follow-up source
identity. Packet inspection verifies 113 file hashes and retains module/config
pins. `runtime/classic.py` maps genuine native model/reconcile/propose/Execute/
guarded Apply/fresh Verify/Audit calls. Apply needs the external exact reviewed
digest. The packet's policy-prepared synthetic smoke demonstrates lifecycle
mechanics only; this package does not rerun or rescore it as a live result.

The conventional arm may use an ordinary strong actor with the same released
requirements and tools. No weak artificial baseline is required. Actual generic
Actor setup, Classic live Executor/Verifier integration, whole-trial enforcement
and Government product semantics all remain unverified.

## Evidence and S1 disposition

`evidence/runner-readiness/` preserves native request/result/raw receipts, focused
tests, the independent review and a manifest of this checkpoint. Tests cover
concurrent reservations, no refills, unknown usage, exact binding, closed live gate,
and descendant termination. Earlier preparation evidence remains bound to its
commits, including `e4278c50ddc84daee3858e24c89e6cf5777b5dfe`; its
`preparation-freeze.json` is historical and does not describe these new files or
the subsequently selected resource profile. The readiness manifest owns this
checkpoint; it does not freeze a live protocol or modify historical receipts.

**S1 is not cleared.** Remaining gates: account-supported actual model identity;
an enforced small provider-call boundary enabling real identity/usage/access
probes; effective context and OS access measurement; complete shared live budgets
including descendants/retries/repairs; real conventional and Classic setup/task
smokes; Government native pins and G1-G5; private oracle/rubric freeze and task-6
equivalence mapping. No six-cell comparison is authorized by this checkpoint.

The prior full source Verify remains `incomplete` because its Go-test step timed
out at 600007 ms. No full Go/Verify suite is repeated here. Focused Python checks
and native readiness receipts cannot turn that historical result into a pass.

Sources: [official non-interactive documentation](https://learn.chatgpt.com/docs/non-interactive-mode)
documents explicit config/rules bypass and JSONL events;
[official config reference](https://learn.chatgpt.com/docs/config-file/config-reference)
documents feature controls, project-doc byte limits and provider retry settings.
Installed CLI/schema evidence remains separately hashed; documented options do
not establish account availability or effective Actor isolation.
