# Native integration: r2 stopped; negative handoff complete

Both corrected mechanical cases ended **incomplete** and are stopped. No
resume, Apply, Verify, Audit, stale replay, semantic acceptance, comparison result
or S1 readiness was established. Neither failure proves a product defect.
There is no remaining active execution under this grant. After compaction,
continue only an explicitly assigned new task; do not resume these cases.

## Exact authority, source and history

Overseer accepted r1 at `f35c8209cd8902bff17e69f05ee7716e352ee4b2` and issued
`native-s1-corrected-integration-20261008-r2`. [Exact grant](evidence/native-integration/run-2/authorization-grant.json)
SHA-256 `be78d1156bfe49f46793906d94dfd0ec82ace550cbb107b425d383d45e25870f`
is bound to the canonical Scientist correction pointer and immutable snapshot
SHA-256 `1b2e361ce2a321149a65694b6586bab57e74388dc5486bc2d4f8cbedb2b7a75c`.
The old grant remains product-pin/no-provider provenance; the separate released
Request correction supplies additive execution authority. The original native
SQLite ledger was appended, retaining all three prior rows. Fresh role ledgers
are explicitly bound to [predecessor history](evidence/native-integration/run-2/history.json)
and replace no historical ledger.

Corrected harness source commit: `9b7ee8a1d2ba0a19ea3a717635afca22b1bd51df`;
clean preflight/freeze commit: `5caad2f4004c769ff77974ea64bf8cfbe19dece7`.
[Immutable preflight](public/native-integration-r2-preflight.md) SHA-256
`0109d475f15697ad1025f8bdae524d42fb1a617966959eb75b2b645dc0d41dad`;
[freeze](evidence/native-integration/run-2/preflight-freeze.json) SHA-256
`55c53bf49b006165c3305e470c3258ba1807079511793ffc899e8dd46845d0b9`.
All **82** frozen input hashes were independently checked before the first call
(the reviewer's initial 67-entry count was corrected before closure). No product
or runtime source was changed after that freeze or between the cases.

Product pins remain [Government G5](runtime/government-pin.json), source
`04e225d5caee78c2a198607143863fca1e829750`, executable SHA-256
`12241f325e4af59451e4021b31d9e6f5b829b5de06c94d35eaabfb9d663aa51f`, and
[Classic v0.14.1](runtime/classic-pin.json), held source
`c91363b7ac4decbe87212ff0f588b5451581a152`, runtime build source
`7dbd599c81540c8203a1b7f83afbc335174f4f1f`, executable SHA-256
`2cad55efad64f15d7f57638bea78312918fbf7d181c730f188b9921504da71c4`.
No new Government inspect occurred: byte-identical Constitution/config inputs
allowed reuse of the original inspected digest, explicitly recorded in preparation.

## What was corrected and validated

Classic preparation leaves RecordStore absent and refuses an existing root
without deleting it; both pre-start checks passed. Its main path now supplies
the exact released original grant copy plus correction. Government missing
Evidence identity now preserves bound native queue/run/process receipts and
finalizes the new controller incomplete. Early, bounded stdout/stderr capture
and digest receipts count each failed wrapper before delegate reservation,
with an atomic six-wrapper cap per product. These fixes changed only Scientist
harness code; original r1 evidence and its running Government row remain intact.

Focused regressions passed **102 tests**, followed by **9 authority/controller
tests** after the final exact-grant schema correction. Both exact static input
validations and both additive-budget validations passed before execution.
Tests establish these mechanical guards, not a successful native workflow.

## Actual r2 Government case

One queue process, exit 0, wall **2.280765600 s**, no process stop; queue/job/run
and outer Result are `incomplete`. One native wrapper start, **zero delegate
reservations/launches**, no decision/votes/reviews or accepted candidate.
Hash-bound raw stderr now records `ValueError: delegate argv does not begin with
its bound command` at `government_roles.py:691`, before `_reserve`.
Stderr SHA-256 `5f3b56e9b6f4f85d29fe8f230695413118545da6f58b60351ad57f223dba6bd0`.
This identifies the r2 guard failure; it does not retrospectively identify the
missing raw r1 stderr or prove that r1 had the same cause.

The new controller ended `incomplete`, start `1791415569.282673`, end
`1791415571.6372206`. Its controller receipt and Result preserve the actual
native process; the separate `controller_processes` table has zero rows for
Government. The old r1 controller remains running, unchanged. No resume followed.
See [Result](evidence/native-integration/run-2/government/queue-result.json) and
[independent Government post-run review](public/native-integration-r2-government-postrun.md).

## Actual r2 Classic case

One Execute process, exit **2**, wall **3.359596200 s**, no timeout/stop. Native
stderr is `external runner failed`; the early wrapper trace identifies
`ValueError: Classic Invocation is missing the exact configured scope identity`
in `classic_integration.resolve_role`, called while loading bootstrap context.
One wrapper start, **zero delegate reservations/launches**. The original missing
RecordStore prerequisite no longer blocked this call, but Execute did not succeed.

The new controller ended `incomplete`, start `1791415612.6915567`, end
`1791415616.4741743`, within its original 180-second window; one
`execute/incomplete` process row. No external Execute approval was issued.
Apply, fresh Verify, Audit and replay lacked prerequisites and were not attempted.
See [Result](evidence/native-integration/run-2/classic/flow-result.json) and
[independent Classic post-run review](public/native-integration-r2-classic-postrun.md).

## Kumulative consumption and remaining limits

| Product | Native r1 + r2 = total / r2 cumulative ceiling | Wrapper r1 + r2 = total | Delegates | Reserved seconds r1 + r2 = total / ceiling |
|---|---:|---:|---:|---:|
| Government | 2 + 1 = 3 / 4 | 1 + 1 = 2 | 0 | 300 + 150 = 450 / 600 |
| Classic | 1 + 1 = 2 / 6 | 0 + 1 = 1 | 0 | 150 + 150 = 300 / 900 |
| Combined | 3 + 2 = 5 / 10 | 1 + 2 = 3 | 0 | 450 + 300 = 750 / 1500 |

R2 used one of at most six new wrapper invocations per product. Products ran
sequentially; role parallelism maximum 2. Native process ceilings stayed 38 s,
controller windows Government 38 s / Classic 180 s, with no clock reset/refill.
Each 150-second reservation already covers the controller plus up to two bounded
role sessions and cleanup; delegate wall is not charged again on top. Arbitrary
OS-descendant aggregate wall time remains unknown/null. Native provider/token
telemetry remains null/unknown, not measured zero. No sixth real Actor/model
session, provider call, metadata tree, study cell, purchase, push or publication.
Historical five real Actor starts, known tokens 53,331 and unknown total remain
unchanged. Unspent numerical quota permits no repair/relaunch of these cases.

[Quota and ledger summary](evidence/native-integration/run-2/quota-and-ledger-summary.json)
contains all five native claims/receipts and terminal new-controller states.
[External archive](evidence/native-integration/run-2/external-snapshots.json) binds
86 original/copy pairs, including raw diagnostics, reports and closed SQLite
snapshots. Actor checkouts are excluded; their exact bases remain bound. The r1
archive stays immutable; only its original live native-budget database received
the authorized appended rows. The separate metadata package at
`498faa073c91acf9d66d375a061d37b851cca355` was not rerun or changed.

## Smallest next work, not performed here

1. Reconcile Government delegate construction with the exact command/argv schema:
   the frozen slots provide `argv`, while the observed guard requires a matching
   `command`. Add a regression using those exact prepared slots before any newly
   authorized case. Do not repair r1/r2 ledgers or infer r1's missing error.
2. Compare Classic's actual pinned native Invocation scope fields with the bridge
   resolver contract, preserving strict identity checks. The current trace locates
   the mismatch but does not justify guessing or relaxing the scope binding.
3. Real runner/tool access and semantic independent evaluation remain separate
   unresolved gates. These deterministic failures do not establish S1 readiness.

All granted execution is stopped. This package closes the bounded negative
attempts and their evidence; it does not close native integration.
