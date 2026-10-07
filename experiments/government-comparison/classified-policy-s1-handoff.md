# Scientist: classified metadata result and native adapter preparation

Base `3be653384e1b83d8cbdab79a87f487a3ffdde050`, feature branch
`codex/government-scientist`. Overseer chat
`01a11367-a781-7683-a20f-46e12614dcb4` authorized the existing two workstreams;
its continuation added no resources. This handoff separates a consumed metadata
session from pure adapter mechanics. Neither supplies autonomy or quality evidence.

## Classified metadata: valid negative result

Allocation `local-policy-read-classified-20261007` was durably reserved before
one controlled app-server process tree. The [frozen request](evidence/policy-read-classified/run-1/request.json),
[freeze](evidence/policy-read-classified/run-1/freeze.json),
[reservation](evidence/policy-read-classified/run-1/reservation.json),
[launch claim](evidence/policy-read-classified/run-1/launch-claimed-once.json) and
[native-start marker](evidence/policy-read-classified/run-1/started-once.json)
bind the exact code, methods, inputs and prior consumption. The independent
[client review](public/policy-classified-review.md) preceded reservation and start.
Seventeen pure tests passed on the final client. Earlier unexecuted pre-review
candidate copies remain separately labelled; they are not launch evidence.

Archival limitation: the reviewer edited the preflight note while adding the
post-run review. Its original bytes were not preserved and could not be recovered
at the reservation-bound SHA-256 `e8d07fb7214c61739c1944d14d0fef05ef97fc136bfed1c4c0848631f1c7acb1`.
The reservation command verified that exact file hash before launch, and the chat
records the independent clearance. The linked review is the later revision;
the packet cannot independently reproduce the original preflight note's bytes.
No replacement text is presented as that original artifact.

The [sanitized result](evidence/policy-read-classified/run-1/sanitized-result.json)
records `initialize` and `initialized` only. Before either read RPC, the client
received `remoteControl/status/changed`, classified against the exact binary's
frozen enum as `RemoteControl/status/changedNotification`. This is outside the
four allowed notification exceptions, so it stopped without answering. Only
method and class were retained; params, server IDs and notification contents
were discarded. The earlier session's discarded identifier remains unknown:
this new observation does not retrospectively classify it.

| Observation | Actual value |
|---|---|
| Native binary | Codex CLI 0.160.1; SHA-256 `3b8f6e33caa75f232558a3cf76ff9b87bb5ef6dbcf4996372f24e55c78b1b916` |
| Native exit / client exit | `0` / `1` |
| Native session / controlled tree elapsed | `0.181643800 s` / `0.527709300 s` |
| Config / requirements reads sent | `0` / `0` |
| Stop | `non-discardable-notification` |
| Raw bytes consumed then discarded / native stderr bytes | `420` / `0` |
| Retry / extra allocation trees | `0` / `0` |

The request preserved the original Actor cwd and all fifteen inline config pairs
in order. The documented app-server route still omits the historical exec-only
ignore-user-config/ignore-rules, sandbox/model and ephemeral controls. Ordinary
configuration layers may participate, so this route is not an equivalent exec
policy observation. No rights, global settings, authentication, sandbox setup,
command, filesystem or write RPC was added. Actor/model RPCs were excluded.

Only correctly shaped `warning`, `configWarning`, `deprecationNotice`, and
`windows/worldWritableWarning` could be discarded; all other methods, every
server Request, malformed input, auth activity and RPC error stopped the session.
The total notification cap was 16. Non-deprecation warnings would make any config
result provisional; none occurred, and no config result was obtained. A false
`provisionalConfig` therefore supplies no policy or access proof. Warning text
was never interpreted. Native internal authentication/service behavior was not
traced and cannot be inferred from these protocol observations.

The final client has a 33-second active RPC deadline. The existing Windows Job
controller used a tighter 38-second whole-tree deadline, leaving the controller's
up-to-two five-second cleanup waits plus margin inside the granted 50-second
inner and 60-second outer ceilings. This also encloses final draining and receipt
writing. The queue remains 32 frames and raw ingestion threshold 4,000,000 bytes,
with bounded transient line buffers rather than an exact peak-RAM guarantee.

The newly authorized one-tree allocation is consumed. The six earlier CLI
metadata calls and first metadata tree remain consumed. Five historical Actor
starts, known input-plus-output subtotal **53,331**, and unknown historical total
remain unchanged. No sixth Actor session, study cell, rejected read retry or
additional diagnostic loop was authorized or run.

## Adapter mechanics and accepted product identity

The [mechanics table](public/s1-adapter-mechanics.md) names bound inputs, actual
supported native commands, implemented translation and remaining S1 evidence.
The [Government helper](runtime/government.py) validates exact Request/file
bindings, produces review-only queue/resume argv, and normalizes native queue,
run, role, vote, decision and promotion receipts. These helpers cannot execute a
product. The dispatcher continues to return `readiness_gap` before reserving an
Actor session. An exit-zero native queue status never means independent acceptance.

During this assignment Overseer accepted final G5 source
`04e225d5caee78c2a198607143863fca1e829750`. The final
[Government pin](runtime/government-pin.json) binds its original handoff SHA-256
`b83103dc1f0c9d9985bfe540692869754abbd40249809e55c6aebc4a9d24417b`
and Windows binary SHA-256
`12241f325e4af59451e4021b31d9e6f5b829b5de06c94d35eaabfb9d663aa51f`.
Both actual files were hashed locally. The prior `ce021ce…` identity remains
history. Its lease finding is closed by Overseer's acceptance; that acceptance
does not establish live-model quality, autonomy or S1 readiness. The original
Worker handoff's earlier pending-status text is preserved; the later acceptance
is separately sourced to Overseer's message.

Classic remains source `c91363b7ac4decbe87212ff0f588b5451581a152` / v0.14.1,
using existing `runtime/classic.py` and its immutable packet. The later `8927704`
candidate is excluded. No native product or Actor was invoked for adapter tests.
The independent [adapter review](public/s1-adapter-review.md) and focused
[mechanical validation](evidence/s1-adapter-mechanics/validation.json) describe
the implemented scope and remaining limitations. The final focused gate passed
23 tests, with its exact command, source hashes and [log](evidence/s1-adapter-mechanics/test.log)
preserved. It covers translation, a real temporary Authority and shared ledger,
grant exhaustion, dispatch-scoped token checkpoints, replay prevention and timeout capping; it starts no native
product, Actor or provider. The separate frozen metadata client has 17 passing
classification tests and was not rerun after its single consumed session.
The earlier 22-test gate is archived separately under
`evidence/s1-adapter-mechanics/before-token-threshold-fix`; it predates the final
token-checkpoint fix and is not evidence for that later source.

## Smallest outstanding S1 step

Government's local journal does not reserve roles in the common study ledger.
Its configurable RunnerSpec and documented agentexec JSON invocation/response
protocol provide the integration boundary. The importable Scientist role helper
now reserves each unique delegated invocation before effects, enforces both the
common limits and the bound operator session ceiling, and preserves unknown
usage and ambiguous interruptions. Delegate timeouts are capped by remaining
outer Request, operator/role expiry, session, trial and task wall limits. The
operator's retrospective token checkpoint covers the current dispatch's outer
attempt and nested roles; common token limits remain trial-wide. This checkpoint
cannot prevent in-flight overshoot or supply missing usage. Its tests construct a real temporary
`dispatch.Authority` and use a provider-free delegate. The CLI remains disabled:
an acyclic bootstrap for that Authority and native queue wiring are still open.
The present outer-dispatch precondition conservatively books an extra Actor
attempt; native integration must give the controller separate bookkeeping and
charge every real role, rather than treating a queue as one Actor. Bind this
integration and one released S1 task's project/runtime/backlog inputs to exact
operator authority and the accepted pin before a jointly authorized native
setup/access smoke. Missing provider usage stays unknown; no synthetic usage or
fixture replaces live evidence.

For policy diagnosis the immediate obstacle is now a classified, disallowed
notification before reads. Any future inbound-policy exception or fresh metadata
session requires a new explicit coordinator decision; this package makes neither
change. It identifies no cause of the earlier exec rejection and no justified
permission change.

The [delivery validation](evidence/policy-read-classified/run-1/validation.json)
binds this package, frozen metadata inputs, process receipts, preserved historical
evidence and four unchanged ledgers. Q1, fixtures, rubric and common budgets are
unchanged. Scientist/reviewer preparation active-time, token and cost totals are
unavailable, not zero. Hold for the joint S1 decision after this finite handoff.
