# Finite S1 dispatch and accounting

The strict-profile behavior below records the preceding S1 package. The current
common admission semantics and append-only successor authority are specified in
[measurement-profile-v2.md](measurement-profile-v2.md). S1 remains open; this
update creates context-session drafts only and issues no live grant.

This candidate connects `harness.dispatch` → `adapter.handle` → `dispatch` →
`Ledger`/`process.bounded`/the pinned `runner` argv. It does not clear S1 or
authorize a model call. The package executes only deterministic external Python
programs, using synthetic counters. Classic/Government native execution and resume
remain `readiness_gap` before reservation. Classic stays v0.14.1 with runtime
7dbd599 / held c91363b; Architect 8927704 is not a study candidate. The Coordinator
accepted Government G2 at `2b604d1aeabd8cef49fb19177cc5d29cfa3b1493` during this
package. Its named `handoff-2b604d1.json`, docs/government.md and example README
were read as a public intermediate interface: `government --action run` with
explicit `--runtime`/`--write` and one Writer. This is deterministic native
mechanics evidence, not a final StudyGovernmentVersion freeze or real model
quality. No Worker binary or trial was executed here; stale final-native-trial.log
was not used. G3–G5 and common-ledger native integration still remain. Private
assessment remains Scientist-owned.

## Operator authority and public entrypoint

`mode=live` alone raises a gate error. The operator must supply four arguments
independent of the public Request: grant path + exact SHA-256 and protocol path +
exact SHA-256. Live additionally requires the Boolean `--allow-live`. No approved
live grant is created by this package. The historical three diagnostic starts
remain exhausted and their databases are untouched.

```powershell
python experiments/government-comparison/runtime/adapter.py `
  --request ABSOLUTE_REQUEST --result ABSOLUTE_RESULT `
  --grant ABSOLUTE_GRANT --grant-sha256 EXACT_GRANT_SHA `
  --protocol ABSOLUTE_PROTOCOL --protocol-sha256 EXACT_PROTOCOL_SHA
```

Omitting `--allow-live` permits only mechanical authority. The Python harness API
accepts these same keyword arguments and passes them to the same adapter path;
it does not run a second Actor. The old fixture `probe` CLI remains a separate
zero-Actor historical readiness interface.

The grant is schema 1, `status=approved`, purpose `s1-mechanics` or
`s1-public-smoke`, with matching mode/trialId, finite `notBefore`/`expiresAt`,
protocol/profile/runner SHA, an external ledgerPath, a dedicated external existing
resultDirectory, maxActorSessions, maxSessionWallSeconds (≤180), and
retrospectiveTokenThreshold (≤10000). `authorizedRequests` contains precisely one
entry per dispatchId, each binding `initialRequestSha256` and `executionSha256`.
The execution digest is SHA-256 of sorted compact JSON with only `operation`
removed; the initial digest binds the exact original file bytes. No grant is
inferred from the Request. Approval is trusted operator input, not cryptographic
identity authentication.

The protocol has `status=frozen`, matching mode, the unchanged `commonLimits`,
`runtimeSourceSha256` from `dispatch.runtime_pins()`, wrapperPythonSha256 and
runnerPinSha256. Mechanical runnerPinSha256 binds `mechanical_pin()` (current
Python path/hash and known fixture hash). The external fixture must have exactly
those bytes; arbitrary executable/argv routing is unavailable. Native runnerPin
binds `runner.inspect(runnerExecutable)`, the exact requested model `gpt-6.1-sol`
and reasoning `high`. Native protocol also includes `authenticatedModelListing`
{path,sha256}; its JSON contents must report successful ChatGPT account metadata,
no RPC/process error, and exactly the requested model/high entry. The listing
can be cached and is not a serving-model receipt. Missing serving metadata stays
null. No aliases or substitutions are introduced. Live grants must explicitly
accept every `dispatch.LIVE_GAPS` value, including unproven Unified Exec/tool
suppression. All arms must use the same selected profile.

Public dispatch Requests add dispatchId, purpose, wallSeconds, task {id,card},
prompt {path,sha256}, releasedInputs, the exact shared limits, and an external
actorRepository/evidenceDirectory. The task card and prompt must be members of
releasedInputs. Releases cannot reside in the mutable Actor, evidence, Result or
authority roots. The wrapper captures/hash-checks input bytes once, sends the
captured prompt to stdin, and retains exact input snapshots. Source paths remain
same-user files: a later Actor tool read is not guaranteed immutable by this
wrapper. Actor-owned files belong to the explicitly pinned starting repository,
not to the external release list. Native starts require its clean full baseCommit,
purpose setup/context-access and a protocol-frozen toolPolicy (`forbidden` or
`ordinary-tools`). Ordinary tools are needed by the later artificial access
probe; their rights are cooperative, not sandbox-proven. Mechanical Requests also
name mechanicalFixture and one of success/no_usage/sleep/child. Those fixtures
are not implementations of any product arm.

## Durable lifecycle

An atomic SQLite reservation consumes one session and stores the original
Request/command. All harness-launched setup, task, review, child and repair roles
use this same ledger and need their own authorized Request. Transport/semantic
retry loops are absent. Purpose repair additionally checks the configured repair
round count. Native unreported child/model calls remain an explicit observability
gap; their absence is not proven by a feature flag or wrapper session count.

The launcher claims `reserved` → `launching` before releasing a target inside the
Windows Job. Raw stdout/stderr and process receipt are retained. Usage observations
never decrease; input+output tokens are counted once, cache/reasoning retained raw.
Agent turns never populate the legacy provider-turn ledger column. Native
providerRequests remains null, so the current strict profile blocks another
reservation after such a session even when token usage is known.

Completion and Result are stored in the same SQLite transaction. The external
Result is an atomic, derived copy confined to the grant's Result directory.
`resume` reuses the original execution identity and never launches or reserves:

- A stored terminal Result is replayed with the resume Request hash and original
  initialRequestSha256.
- A reserved booking is atomically closed incomplete/consumed with unknown usage.
  The launcher cannot subsequently win its reservation claim. An old interrupted
  `recovering` phase can also close without launch.
- A launching record with a complete, internally consistent process/raw-log/input
  receipt is finalized without relaunch or a second counter charge.
- A launching record without that receipt stays blocked and active. No timeout
  or assumed-dead process releases its capacity. Explicit termination proof and
  a bounded follow-up recovery change are required; none is fabricated here.

`stop` persists a shared trial stop; running processes observe it on the 25ms
process poll and retain their terminal receipt. Expiry prevents a new start but
does not disable stop/resume. Wall bound is the minimum of session, remaining
task/trial time and grant validity. Human-time entries stop active observers when
their cumulative reported cap is reached. Known token/request subtotals stop
retrospectively, including across active parallel sessions; no in-flight provider
reservation or exact hard token ceiling is claimed. Wall deadlines use host clock
for aggregate age and monotonic time per process; poll/scheduling latency and
remote provider work are outside an exact instantaneous cutoff.

Raw receipts are local hash-bound records under the same OS identity, not signed
or tamper-proof attestations. Missing/ambiguous evidence stops progress. Grant,
protocol, profile and ledger path cannot be replaced to refill a bound trial.
This finite package has no grant-extension service: later staged authorization
must explicitly preserve cumulative accounting, never start a fresh budget
implicitly. No public smoke is scored as an autonomous study result.

## Mechanical verification and next request

Run only the bounded mechanics entrypoint, with a new evidence directory:

```powershell
python experiments/government-comparison/runtime/verify_s1.py --evidence ABSOLUTE_NEW_EVIDENCE_DIRECTORY
```

It runs only dispatcher/context/runtime risk tests, retains an archive and hash
manifest of external synthetic records, and compares the old ledger hashes. It
does not run the Go suite, six cells, native product binaries or model inference.
See the candidate report and [enforcement matrix](dispatch-enforcement.md) for
observed boundaries and independent review. [The next resource request](s1-next-request.json)
is proposed only: one context session, then hold. Profile correction is an
explicit Coordinator decision required before any multi-session native sequence.
The exact minimal semantic proposal is in
[s1-profile-correction-proposal.md](s1-profile-correction-proposal.md).
