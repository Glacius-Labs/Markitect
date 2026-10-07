# Native integration: r2 correction in progress; r1 negative closure retained

Active resumption point: Overseer accepted r1 at
`f35c8209cd8902bff17e69f05ee7716e352ee4b2` and issued
`native-s1-corrected-integration-20261008-r2`. Its exact envelope is
`evidence/native-integration/run-2/authorization-grant.json`; external immutable
snapshot and envelope are under `native-metadata-fixtures-20261008-r2`.
Only the four confirmed harness corrections, focused gates/review, clean source
commit/freeze, then one corrected case per held product are active. No new
native starts have occurred at this checkpoint. Static disposable fixtures and
explicit predecessor history are prepared. Source ownership: case_contracts
Classic prep; s1_adapter Government translation/finalization; tools_source_review
additive budget/controller validation; root early diagnostics and parent driver.
Native limits are additive +2 Government/+5 Classic, six wrapper invocations
per product, +300/+750 reserved seconds, original deadlines and no provider.
Next: finish source review/gates, release exact runtime/Authority bindings, commit
and freeze; execute products sequentially and stop each at its first failure.
After compaction continue this active r2 task, not the r1 stop below. Historical
r1 evidence and its open Government controller must remain unchanged.

## Historical r1 closure

**The integration attempt failed. Both positive cases are incomplete. No
resume/replay, S1 readiness, comparison result, semantic or human acceptance,
or product fault was established.** Native work has stopped. Failed receipts,
consumed reservations and unfinished Government bookkeeping are preserved.
This handoff authorizes no further attempt.

## Exact candidate and frozen inputs

Harness base: `0e62d5f1469d772804d35f2832efe0fa5679389d`. The separate metadata
package is committed at `498faa073c91acf9d66d375a061d37b851cca355`; see
[its handoff](disabled-status-metadata-handoff.md). Its single read is consumed
and must not be repeated. The executed native harness candidate is identified
by the exact file hashes in the freezes below. The delivery commit preserves
those bytes; it was not itself the executed source revision.

| Product | Held source and executed binary |
|---|---|
| Government G5 | Source `04e225d5caee78c2a198607143863fca1e829750`; binary SHA-256 `12241f325e4af59451e4021b31d9e6f5b829b5de06c94d35eaabfb9d663aa51f` |
| Classic v0.14.1 | Held source `c91363b7ac4decbe87212ff0f588b5451581a152`; runtime built from `7dbd599c81540c8203a1b7f83afbc335174f4f1f`; binary SHA-256 `2cad55efad64f15d7f57638bea78312918fbf7d181c730f188b9921504da71c4` |

[Government pin](runtime/government-pin.json) binds the accepted Worker handoff
SHA-256 `b83103dc1f0c9d9985bfe540692869754abbd40249809e55c6aebc4a9d24417b`.
[Classic pin](runtime/classic-pin.json) binds the verified 113-file release packet.
Government Actor BASE: `fb6bfc1c6eefa799606dc19cb990bad139f34662`;
Classic Actor BASE: `20c0b4c85135bc5d76fe4d337ff5450b8d7178bf`. Both remain clean.

- Allocation `native-s1-integration-fixtures-20261008`: [source grant](evidence/native-integration/run-1/authorization-grant.json), SHA-256 `b917f5a5eb99f0a607fd282a81acdf7b085e6a1dba14f89c8d0c32f349c82c3e`. Requests bind the byte-identical external released copy.
- Immutable [preflight review](public/native-integration-preflight.md), SHA-256 `b2088a9c1caa4a511273666d223f02fc25c7b182565e8998040cf04598db10cc`.
- [Preflight freeze](evidence/native-integration/run-1/preflight-freeze.json), SHA-256 `746df37019541ffa7c395afc5b7f6189f93d5b7b75a540f7b5d1431e015b2c10`, binds 77 exact inputs: sources, binary/Python pins, outer Request/Authority/runtime bindings and current-3 preparations. Earlier preparations remain historical.
- [Same-booking recovery freeze](evidence/native-integration/run-1/classic-recovery-freeze.json), SHA-256 `a57987a790b6568482666718e76ae60d61d7f7dda2bc19e788e334f1876c662d`, separately binds the narrow helper, its review and the original preflight freeze. Original frozen source was not repaired afterward.

## Government: two native starts, incomplete queue

One Constitution inspect exited 0 (0.453055 s), then one queue call exited 0
(1.193431 s). Inspect output was YAML; its initial JSON post-parser failure was
resolved from the same saved bytes without a second invocation. Inspect is not
positive execution evidence.

The native queue result is `incomplete`, stage `execute`, error `external runner
failed`, native `actorStarts=1`, repairs 0, in-flight Actors 0. This means **one
failed native wrapper attempt, zero common-ledger delegate reservations and
zero delegate launches**; no real model/Actor session. No evidence identity,
decision, votes, reviews or accepted candidate exists.

Only wrapper stderr digest
`5a1eb70373ac4e071278441cdec844ec0cd01c8ff64094d428c932447d563477`
was retained; matching raw wrapper stderr/private log is missing. The cause is
unknown. The outer translator additionally reports `Government evidence
identity missing`, `receipts=[]`, `usage=null`. That secondary gap does not
explain the native wrapper failure.

The OS process exited, but the common controller row remains `running`, its
Actor attempt null, with no controller-process row/finalization. This is an
unfinished consumed bookkeeping reservation, not a running OS-process claim
or permission to retry. It is preserved unchanged. Resume was not run after
the positive-case blocker. See [Government post-run review](public/native-integration-postrun.md)
and [Result](evidence/native-integration/run-1/government/queue-result.json).

## Classic: one native Execute, incomplete setup

The original helper first failed before any native budget claim/process/role:
its FixtureBudget used the repository grant path while the Request bound the
external released copy. [The original log](evidence/native-integration/run-1/classic-driver.log)
is preserved. The separately frozen [reviewed recovery](public/classic-prelaunch-recovery-review.md)
used the **same booking**, exact external copy and original controller start
`1791413252.1295412`; no new booking, clock reset or quota refill.

One `controller-execute` ran, exit **2**, wall 0.149563 s, stderr exactly
`statat store.json: The system cannot find the file specified.` It produced no
valid Execute result and started no wrapper/delegate. The configured RecordStore
root was an empty directory; the pinned contract requires an absent root so
initialization creates its marker. The frozen `prepare_protocol_runtime` helper
in `runtime/classic_integration.py` precreates that directory, conflicting with
the prerequisite. Post-run observation alone does not establish who created it
or when. This is fixture setup failure, not evidence of a product defect.

The controller finalized `incomplete` at `1791413406.7649841`, elapsed
154.615612 s within its original 180-second window; one process row
`execute/incomplete`, zero role attempts. Apply lacked successful Execute and
its required external review, so fresh Verify, Audit and replay were not run.
See [Classic post-run review](public/classic-native-postrun.md) and
[Result](evidence/native-integration/run-1/classic/flow-result.json).

## Consumption, original deadlines and unknown usage

The original grant permits per product at most 8 native starts, 12 deterministic
role starts, parallelism 2 and 1200 reserved session seconds, for one positive
case and its directly related resume/replay. Native process deadlines are
38 seconds; Government's controller window is 38 seconds, Classic's 180 seconds.
Neither was reset. Each native start retains 150 reserved seconds: controller
plus at most two concurrent role sessions, including cleanup margin. No refunds
or refills occurred; unspent allocation does not authorize another case/retry.

| Product | Native starts / cap | Native wrappers / role cap | Reserved delegates | Reserved session seconds / cap | Native process wall seconds |
|---|---:|---:|---:|---:|---:|
| Government | 2 / 8 | 1 / 12 | 0 | 300 / 1200 | 1.646486 |
| Classic | 1 / 8 | 0 / 12 | 0 | 150 / 1200 | 0.149563 |
| Combined | 3 / 16 | 1 / 24 | 0 | 450 / 2400 | 1.796049 |

Session reservations are not measured aggregate wall seconds of arbitrary OS
descendants; that metric remains null. Native usage remains null/unknown.
Known fixture provider calls, real Actor sessions and study cells are zero;
null telemetry is not measured zero token usage. The fixed-delegate accounting
path was tested but neither native run reached it.

[Quota/ledger summary](evidence/native-integration/run-1/quota-and-ledger-summary.json)
and [budget snapshot](evidence/native-integration/run-1/native-budget-snapshot.json)
record all three claims/receipts. [External snapshots](evidence/native-integration/run-1/external-snapshots.json)
archive 112 files including closed SQLite snapshots, available raw logs, native
reports, released inputs and preparations. Actor checkouts/binaries are not
copied; exact bases and hashes are bound separately.

## Validation and smallest follow-up

Source changes implement acyclic Authority/bootstrap binding, nullable
controller bookings, reserve-before-delegate guards, separate process receipts,
Government acceptance checks, ordered Classic actions and narrowly authorized
fixed-delegate accounting. [Mechanics](public/native-integration-mechanics.md)
describes the source boundary. Captured focused gates passed 88 tests, then
34 affected tests after static API/deduplication corrections, then 23 final
accounting/gate tests. These validate mechanics, not positive native workflows.
No broader product suite or new probe ran during closure.

[Independent delivery validation](evidence/native-integration/run-1/delivery-validation.json),
SHA-256 `ac5e68e7ca79d178ba34a7ed982ad9c3f2d09b6f0d784c5d82a719a240ab667e`,
verified 77/77 freeze inputs, 3/3 recovery inputs, 112/112 external originals
and copies, read-only ledger counts, binary pins and clean Actor/product bases.
It reuses prior B validation's four historical-ledger hashes rather than
claiming a fresh reread. Preflight and both post-run reviews remain immutable.
Historical five actual Actor starts, known token subtotal 53,331 and unknown
total remain unchanged.

Smallest follow-up for Overseer reconciliation, **not executed here**:

1. Government: arrange digest-bound raw wrapper stderr capture before any newly
   authorized attempt, then diagnose the actual error. Preserve native failure
   receipts and finalize future incomplete controllers even when evidence-identity
   translation fails; never retroactively green-finalize this historical row.
2. Classic: align preparation with the absent RecordStore-root contract and the
   main helper's budget grant path with the exact released Request binding before
   any newly authorized case. The recovery did not repair the frozen main helper.
3. Keep real runner file/tool access separate: it remains unproven, the historical
   exec denial cause remains unknown, and there is no sixth model start.

No real Actor/provider/study start, push, PR, release, global configuration or
permission change occurred. Documentation/evidence closure is complete; native
integration and S1 remain open. Stop here.
