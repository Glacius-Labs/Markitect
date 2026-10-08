# S1 adapter mechanics and common-runner readiness

Current source/evidence reconciliation: **2026-10-08**, based on accepted
Classic R3 delivery `cd537ac11765097cb46e9326f295d74059e827a3` (harness source
`9b7ee8a1d2ba0a19ea3a717635afca22b1bd51df`) and Government R7 delivery
`91bb8cf7b3ae242adc02e7f466bf17ca9120a1c1`. **S1 remains open. All six study
cells are NOT RUN.** This document grants no execution.

The earlier matrix's statements that the wrapper CLI was disabled, an acyclic
Authority bootstrap was absent, and neither native arm was wired are superseded
by the bounded fixture evidence below. Its exact prior bytes are retained in
[the dated historical copy](../evidence/s1-common-runner-readiness-plan-20261008/historical-public/s1-adapter-mechanics-before-20261008.md).
Older [runner readiness](runner-readiness.md), [dispatch](s1-dispatch.md), and
[classified metadata handoff](../classified-policy-s1-handoff.md) describe their
own dated checkpoints; their exhausted allocations and raw evidence are unchanged.
The [current common-runner plan](s1-common-runner-readiness-plan-20261008.md)
owns the proposed next operation and its remaining authority boundary.

## Evidence matrix

| Boundary | Conventional / Oldschool | Classic | Government |
|---|---|---|---|
| Native product capability | Ordinary engineering tools and the [Oldschool method](oldschool-baseline-binding-v1.md) are the reference; no Markitect-specific artifact is required. The permanent Oldschool chat is an engineering owner, not a fresh trial Actor. | Held study source `c91363b7ac4decbe87212ff0f588b5451581a152`; published v0.14.1 binary/runtime source `7dbd599c81540c8203a1b7f83afbc335174f4f1f`, binary SHA `2cad55efad64f15d7f57638bea78312918fbf7d181c730f188b9921504da71c4`. No later P1 adoption. | Accepted G5 source `04e225d5caee78c2a198607143863fca1e829750`, binary SHA `12241f325e4af59451e4021b31d9e6f5b829b5de06c94d35eaabfb9d663aa51f`. Native queue, roles, review/vote, decision and promotion interfaces exist. |
| Shared wrapper / Authority / ledger | Common `harness` → `adapter` → `dispatch.Authority` → `Ledger` / bounded process infrastructure exists. The real CLI route is narrowly limited to setup/context-access with a separate approved grant. A common general coding-task path is still absent. | Acyclic controller bootstrap and CLI role middleware are implemented. Native controller and nested role reservations are distinct, with exact scope, pin, invocation and source-grant bindings. These paths currently admit mechanical `nativeFixture*` allocations, not real model delegates. | The same bootstrap and ledger path is implemented. Static RunnerSpec pins avoid the runtime / outer-Request digest cycle. CLI middleware loads the outer Authority from the bootstrap, reserves each unique role before its delegate, and retains process/role receipts. Its native admission is mechanical fixture-only. |
| Actual deterministic integration | Synthetic harness infrastructure and prior public sentinel diagnostics are neither a Conventional study cell nor a measured Oldschool reference. | R3: Execute `planned`; exact independent external review; Apply `materialized-unverified`; fresh Verify `passed`; Audit `complete` without findings or next steps; stale Apply `refused`, exit 2, without second writes. Five native calls, three fixed roles; outer controller completed. | R7: native AND outer Queue completed; run `accepted-scoped` / `complete`; real fresh `inventory-overflow` Go check exited 0; configured technical review, final Ressort assent, decision and promotion were bound. The associated terminal Resume/Replay completed with the same queue, report, candidate/evidence/decision and promotion, and no new roles. |
| Recovery meaning | No real coding-work interruption/recovery proof. | The stale Apply refusal demonstrates a guard against reapplying an obsolete plan. It is not an interrupted real Actor recovery trial. | R7 resumed a queue that was already complete. It proves terminal idempotency and readback, not continuation of interrupted Agent work. Earlier R4/R6 incomplete results remain incomplete. |
| Actual provider / general tools | The common runner has produced responses, but the fifth real Actor failed both authorized file reads at CreateProcess policy admission despite process exit 0. No successful common real coding path is established. | Deterministic Python role delegates exercise the protocol, not Codex/ChatGPT access, writing, build/test tools or engineering judgment. | Deterministic Python executor/reviewer/voter delegates exercise the protocol, not independent model review, real tool access or engineering judgment. The real Go check is a native product check, not an Actor's build/test action. |
| Semantic and human acceptance | No measured reference value or comparative difference. | No real-task engineering-quality or human-acceptance result. | No real-task engineering-quality or human-acceptance result; `accepted-scoped` is a native state, not independent acceptance. |

[Classic R3 flow](../evidence/native-integration/run-3/classic/flow-result.json)
and its [additive accounting](../evidence/native-integration/run-3/quota-and-ledger-summary.json)
retain the original results, including the expected stale replay refusal.
[Government R7 summary](../evidence/government-check-receipt-native-20261008-r7/terminal-summary.json),
[Resume readback](../evidence/government-check-receipt-native-20261008-r7/government/resume-final-result.json),
and [independent post-review](../evidence/government-check-receipt-native-20261008-r7/independent-postreview.md)
retain the completed fixture evidence. Accepted delivery/source identifiers above
refer to harness evidence; they do not replace either product's pinned source.

## Implemented bindings and closed launch paths

`government.bind_request` binds accepted handoff/binary/source, Government YAML
and Order inside the exact Actor repository, and released runtime/backlog bytes.
Queue and resume argv construction remains a pure planning operation until the
separately authorized controller launches it. `government.translate_queue_result`
checks exact Request bytes, immutable queue reports, run-id/path/digests, configured
roles, full four-field scope identities, cabinet/mandates, candidate/evidence/round,
votes, decision and promotion receipts. The corrected fresh-check consumer binds
native `GateResult.Tool` to valid configured `Check.run[0]`, with exact names/count,
zero exit and elapsed/timeout checks. Neither translator supplies independent
semantic acceptance.

`dispatch.dispatch` routes Government to `dispatch_government`, which requires
mechanical fixture authority; Classic still returns `readiness_gap` there. The
accepted Classic R3 driver explicitly calls `begin_classic_controller`,
`run_classic_step` and `finalize_classic_controller` outside that regular route. `native_controller.validate_native_fixture_grant`, bootstrap
loading and delegate admission bind the allowed original fixture authorization,
fixed Python/delegate hashes, exact configured scope/role, and successor profile.
Government's closed R5/R6/R7 selectors additionally require their own exact
`nativeFixtureR5Grant` / `nativeFixtureR6Grant` / `nativeFixtureR7Grant` and dispatch
identity; older correction/R3/R4 gates retain their separate grant identities.
They are not generic permission to launch new cases. All consumed native grants
are closed; changing a Request's mode, delegate command or profile cannot admit a
real Actor. Source function references and hashes are retained in the new plan's
[source evidence](../evidence/s1-common-runner-readiness-plan-20261008/source-evidence.json).

The minimal missing bridge is a separately reviewed real delegate path: convert
one bound native Invocation into one fresh common Actor context, translate its
actual response into the required native schema, preserve exact pins and unique
pre-launch reservations, and attach actual events, tools and usage to the same
trial/task ledger. Conventional needs the equivalent ordinary task/review path
through that common runner. This must retain independent reviewers and product
mutation gates, rather than inserting a live command into fixture authorization.
No such implementation is part of this documentation package.

## Remaining common readiness

The failed fifth Actor and the three consumed policy-read sessions are
[consolidated in the plan](s1-common-runner-readiness-plan-20261008.md#historical-runner-barrier).
The last app-server read included user configuration and omitted exec-only
ignore-user-config/ignore-rules controls. Its reported configuration does not
explain the earlier exec's effective policy. The cause of both denials is unknown.
Reading alone would not demonstrate writing, builds or tests.

The adopted [measurement profile v2](measurement-profile-v2.md) retains unknown
provider requests/internal retries as unknown and tokens as retrospective. A
finished attempt with unknown tokens still blocks subsequent admission; exhausted
history cannot be reset by a new path, database or trial identity. Native controller
bookings and nested Actor attempts are separate. The fixture ledgers prove those
mechanics, not complete real-descendant accounting or hard in-flight token caps.

Before any study cell: one common verified real runner/tool boundary for all
three arms, source-bound real delegate integration, equal context and access,
finite authority/accounting, released setup/task inputs, and independent private
assessment/task-equivalence remain required. Desired `gpt-6.1-sol/high`, prior
model-list advertisement, and observable serving/provider identity are distinct.
Unavailable serving metadata remains null; it is not an impossible prerequisite
or a model self-report. No product tests or experimental invocation were run for
this reconciliation. S1 and all six cells remain open / NOT RUN.
