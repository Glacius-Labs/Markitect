# Runner and common adapter readiness

This is a finite readiness package, not a comparative result. Live trials: **0**.
Actual external runner start attempts: **2 of the authorized maximum 2**, remaining **0**;
started agent turns: **1**, completed **0**. Attempt 1 failed configuration loading;
attempt 2 reached the service and received an explicit account/model rejection.
No model was substituted and no new authentication, purchase or user-config change occurred.
The common public adapter contract remains v1.1.

## Final authorized diagnostic (attempt 2)

The Overseer explicitly authorized the one remaining start despite attempt 1's
failure and unknown usage. This exception applies only to this tiny diagnostic.
No further Actor, access probe, continuation or retry start is authorized.
The historical attempt-1 section and its frozen evidence remain unchanged below.

`runtime/remaining_probe.py` removed the unsupported built-in-provider definitions
while retaining the authenticated built-in OpenAI provider, explicit
`gpt-6.1-sol/high`, original saved auth, 180-second Windows Job deadline, parallel 1,
one wrapper-started turn, disabled tools/apps/plugins/hooks/subagents and no wrapper
retry or continuation. It changed no endpoint, provider, global config or credential.
The no-retry rule now concerns wrapper launches; unchanged built-in internal
transport defaults were permitted. The official current config reference documents
HTTP retry default 4 and stream retry default 5; the applied values in this binary
and observed internal retry count remain unknown. `automaticRetries=0` in the
process receipt describes wrapper launches only.

Before launch, `features list` parsed the exact corrected settings successfully
using an empty, credential-free CODEX_HOME for that non-Agent subprocess only.
The actual Actor used the original auth location. Read-only account metadata
commands failed on ambient agent scalar settings incompatible with this CLI;
neither `agents.model={}` nor `agents={}` fixed that parsing. The bundled catalog
command bypasses ambient config parsing; its six-model list omitted the requested
model, which alone was not an account rejection. The original preflight-policy
note was incomplete and is corrected in the separate observation, without changing
the preserved raw file.

The actual corrected start exited 1 after **4.57096099993214 seconds**, with
`thread.started`, one `turn.started`, an error, and `turn.failed`. The service's
HTTP 400 error explicitly says the requested model is unsupported when using
Codex with a ChatGPT account. No agent message or token receipt arrived. Resolved
model, provider-request count, internal retries, known token subtotal and aggregate
tokens remain `null`; attempt 1's unknown usage also remains unknown. This is an
account/model rejection for this pinned CLI/auth invocation, not evidence that
the desktop app or a different runner cannot use that model.

Native stderr also reports malformed local skill YAML and failure to decode the
model cache/refreshed catalog because this binary does not recognize reasoning
level `max`. It includes a raw service model catalog. No alternate model was tried.
The skill-loading diagnostic shows that global resource discovery still occurred;
the disabled execution flags and independent public directory do not establish
effective context privacy. No successful model response, access isolation, token
threshold stop or study execution was demonstrated.

Exact new receipts, invoked source snapshots, preflight failures and the durable
two-start ledger snapshot are in `evidence/actual-runner-probe-r2/`, separately
frozen by `actual-runner-probe-r2-freeze.json`. The fixed operator ledger prevents
refilling quota by changing the external directory. S1 stays open. Later study
`commonLimits`, Classic v0.14.1 and the Government readiness gap remain unchanged.

## Historical attempt-1 addendum

After accepting infrastructure checkpoint `8df6085d938a1f73576743c3fa38ba3a5c51dd9a`,
the Overseer explicitly removed the former six-internal-provider-request boundary
for this tiny probe only. The current authorization is at most two fresh starts,
180 seconds per owned process tree, parallel 1, one wrapper-started agent turn,
no continuation/retry/subagents/plugins/hooks or additional runners, and a
retrospective aggregate stop threshold of 10000 reported input-plus-output tokens.
Internal provider requests remain separately unknown. This does not change the
selected later-study `commonLimits` or clear their enforcement gaps.

Scientist actually started the pinned `codex-cli 0.130.0` with explicit
`gpt-6.1-sol/high`, existing saved ChatGPT auth, no API-key environment fallback,
ephemeral/read-only mode, disabled shell/unified execution, memory, apps, plugins,
hooks, goals and subagents, and a new public random sentinel in an independent
Documents root. It was one launch, protected by the existing Windows Job and
180-second deadline; no real study inputs or project work were supplied.

The native CLI exited 1 after approximately 0.18 seconds, before emitting any
JSONL event or starting an agent turn. Its unchanged stderr says that
`model_providers` contains the reserved built-in ID `openai`, which cannot be
overridden. The prospective zero-retry overrides
`model_providers.openai.request_max_retries=0` and
`model_providers.openai.stream_max_retries=0` were therefore incompatible with
this actual runner. This is a demonstrated runtime/configuration failure, not
an account rejection or evidence that the requested model is unsupported.

The failed start consumes attempt 1. Probe 2 is not launched because it was
authorized only after an executable Probe 1 and remaining observed budget.
Resolved provider model, provider usage/request count and effective Actor context/
access remain unknown; no model self-report is used. There is no token receipt,
so the observed-threshold monitor cannot be credited with an actual provider-token
stop. Its lossless raw stderr/stdout, command/config/request binding, process
receipt, durable attempt ledger and source hashes are preserved under
`evidence/actual-runner-probe-r1/`.

Before another real start, the coordinator must settle a supported way to obtain
the required retry behavior from the built-in authenticated provider, or explicitly
adjust that tiny-probe rule. No renamed/custom provider, alternate model, second
retry start or global configuration repair was attempted. The later live dispatcher
also still needs aggregate accounting, concurrency reservations, retry/repair
policy, effective context/access measurement and real setup/task evidence.
Classic v0.14.1 / runtime `7dbd599` / held `c91363b` remains unchanged.

## Decision and enforced boundary

The coordinator selected the existing `commonLimits` in `resource-proposal.json`
unchanged: per task 1200 seconds, 12 Actor sessions, 80 provider calls and 120000
tokens; per trial 7200 seconds, 72 sessions, 480 calls and 720000 tokens; parallel
4, transport retries at most 1, semantic repairs at most 2, active human time 600
seconds. All setup, model creation/upkeep, ministries, descendants and reviews
consume the same profile. Selected limits are not a claim of technical enforcement.

At the historical infrastructure checkpoint, the separate tiny runner authorization permitted at most two fresh sessions, each
180 seconds and six provider calls, parallel 1, no descendants or retries, and a
10000-token aggregate stop threshold as counters arrive. The installed runner's
CLI help and generated app-server schema did not establish a pre-dispatch provider
call limit. `exec` reports `turn.completed` for an agent turn; that event is not
evidence of the number of provider requests inside it. Consequently that checkpoint
launched no provider probe. The addendum above supersedes this former tiny-probe
boundary. The final diagnostic above records the later explicit account/model
rejection; an executable supported identity remains a concrete readiness gap.

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

**S1 is not cleared.** Remaining gates: a supported native runner/account configuration,
account-supported actual model identity; successful real identity/usage/access
probes under the selected probe rules; effective context and OS access measurement; complete shared live budgets
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
