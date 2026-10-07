# Dispatcher enforcement and bounded pilot profile

This is an independent review of the current local dispatcher candidate. Its
25 focused mechanics tests pass, but that does not establish live-provider
enforcement or clear S1. The dispatcher is cooperative same-user software:
hashes and receipts bind recorded bytes, but do not make the workspace
tamper-proof.

## Enforcement boundary

| Resource or claim | Current candidate behavior | Evidence and remaining boundary |
| --- | --- | --- |
| Authority and request binding | Requires an operator-supplied approved grant and frozen protocol with matching digests, pinned source/runtime and runner, unchanged `commonLimits`, finite grant interval, authorized dispatch/execution digest, and exact initial request bytes. Live mode also requires explicit `allow_live is True` and the accepted gap list. | Authority is checked before dispatch; 25 deterministic dispatcher tests pass, including public adapter CLI forwarding. This is local cooperative validation, not a signature or same-user tamper barrier. |
| Actor sessions and retries | Atomic SQLite reservation records one session before launch; the grant cap and common task/trial caps are checked in the same transaction. Failed/ambiguous starts consume the reservation. The wrapper launches once and declares zero automatic wrapper retries. Classic/Government return `readiness_gap` before reservation until native integration exists. | Dispatch ID is unique and resume does not relaunch. This counts wrapper Actor sessions, not provider requests or provider-internal retries. |
| Parallel work | SQLite `BEGIN IMMEDIATE` serializes reservation against the shared parallel-session limit. Active owned work shares the trial stop state. | Concurrent-reservation and concurrent-same-dispatch tests pass. Provider activity inside parallel sessions is not observable as an aggregate and can overshoot. |
| Wall time | Admission checks task/trial deadlines. The active monitor checks cumulative task/trial time, grant expiry, requested session wall, and shared stop; the process wrapper bounds the local process tree and kills it at the deadline. | Focused deadline and shared-stop tests pass, including a local descendant process. This does not bound remote work already accepted by a provider or descendants beyond the wrapper's OS job/group boundary. |
| Human time | Ledger records cumulative reported active seconds and sets the shared stop at the limit. | The dispatcher observes the stop while work is running and blocks later reservations. Human time remains an operator-reported measure. |
| Turns, tokens, and unknown usage | Agent-turn events are parsed separately from provider requests. Token usage is retrospective; the 10,000-token session threshold stops subsequent polling when observed and records any overshoot. Known totals are checked at admission; ended sessions with unknown provider usage block further admission. | No hard in-flight provider-turn/token cap is established. Provider request count and internal retry count remain null/unknown; agent turns must never be reported as provider requests. Parallel calls can overshoot before receipts arrive. |
| Model identity | Live authority validates a hash-bound authenticated listing that advertises the exact requested `gpt-6.1-sol` ID and `high` reasoning, plus the pinned runner. Output records requested ID/reasoning; resolved serving model remains null. | Catalog advertisement can be cached and does not prove the serving model. The candidate must never silently alias or substitute another ID. |
| Tools and context | A frozen protocol chooses `forbidden` or `ordinary-tools`. Event polling and finalization classify tool events; forbidden activity triggers a stop and contamination flag, while ordinary tools are allowed except collaboration-like events. | Detection is reactive and cannot prevent an initial tool action. The selected runner reported `unified_exec=true` despite a requested false setting, so suppression is unproven. Instruction/context privacy and filesystem isolation are not established. |
| Mechanical fixture | Only the pinned Python executable and known fixture bytes are accepted; only the conventional arm can use this deterministic fixture. Arbitrary commands and native product-arm simulation are rejected. | Tests cover deterministic accounting and process recovery; this is harness testing, not evidence of product quality or inference. |
| Inputs and evidence | Validation captures each released input once, checks its digest, and rejects it inside mutable Actor/evidence/result roots or at an authority path. The exact captured prompt bytes feed stdin; copies are retained as evidence and rechecked during finalization. Actor, evidence, and result directories are separate. | The mutable-path and prompt TOCTOU findings are fixed and covered by a focused regression. Evidence remains mutable to the same user; these local records are cooperative, not tamper-proof attestations. |
| Crash recovery | An unlaunched `reserved` or prior `recovering` booking is terminalized as consumed/incomplete in one SQLite transaction; the launcher CAS cannot win after that transaction. A launching booking without terminal receipt stays active/blocked; a stored terminal receipt can be recovered without relaunch. | The recovery-transaction crash gap is fixed and covered by a focused regression. Same-user edits can forge local receipts, so describe them as cooperative records rather than tamper-proof attestations. |

`resource-proposal.json` remains authoritative for shared limits: 1,200 seconds
per task, 7,200 seconds per trial, 12/72 Actor sessions, 80/480 provider turns,
120,000/720,000 provider tokens, four parallel Actors, one transport retry per
call, two semantic repair rounds per task, and 600 active human seconds per
trial. The dispatcher does not silently change those limits. In particular, an
agent turn is not a provider request, and missing provider/retry counts stay
unknown. The proposed alternative observation profile is not adopted; see
[s1-profile-correction-proposal.md](s1-profile-correction-proposal.md). Until
there is an explicit shared-profile decision and a reviewed cumulative-ledger
change, strict unknown-provider admission remains in force. No reset or new
ledger can be used to bypass prior attempts.

## Smallest measurable next request

The three earlier identity-diagnostic starts are exhausted and cannot fund
further work. Keep `commonLimits` unchanged. The next separate resource request
should authorize **one context/access session only**, with a maximum of 180
seconds, one requested agent turn, no wrapper retry/continuation/repair or
subagent, and a 10,000 observed input-plus-output-token stop threshold. Run it
sequentially. That token threshold is retrospective, not a hard cap; retain any
overshoot. Its result is a context/access diagnostic, not an arm smoke or task
result. Stop afterward and require a separate decision and grant for any later
session; this request does not authorize the conventional, Classic, or
Government session.

For a later matched arm pilot, the smallest comparable sub-cap would be one
session per arm, sequentially, with the same public task and captured prompt
bytes, tool policy, requested model ID, reasoning, and result schema. Bind the
requested ID to the exact authenticated listed ID and pinned
executable/config/source digests; require the exact listed ID with `high`,
allow the serving-model field to remain null, and never silently alias or
substitute. Classic and Government remain unreserved readiness gaps until their
native adapters are integrated. Unknown usage after a session blocks the next
reservation; provider-request and retry counts remain unknown. This is a
proposal for a future authorization, not permission for additional sessions or
a six-cell comparison.
