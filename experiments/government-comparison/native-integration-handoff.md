# Native integration R3 terminal: Government negative; Classic mechanics positive

The existing grant `native-s1-contract-corrected-integration-20261008-r3`,
amended before execution by `r3-a1`, has reached its terminal cases. No R3
retry, Government Resume, new case, repair-and-run, or reuse of unused quota
is authorized. This local delivery commit owns the terminal evidence package.
The Scientist slot is released explicitly in this chat after evidence review;
Overseer alone owns the canonical coordination state and its next assignment.

## Source, authority and freeze

Accepted offline predecessor: `f5cd8871b84f45541044110057b59f0bb1ca0804`.
Runtime admission source: `36ac50558eabcb44812b2c85d723c9da5b783e4c`.
Final checkpoint driver: `0f24deb6534075922acdf35ccf8bfcd330cbf24a`.
Independent preflight committed at `d17da61`; source/evidence were clean before
freeze, and the final freeze was committed clean at
`5795069128586977d1be66af6564846e2915c166` before any actual R3 start.
Freeze SHA-256: `acd01a42a7a7b511188479e66222430b5d9d628a4661fe7d7c5ff0bf6f5190ba`.
All 102 frozen input files were rechecked unchanged after both products ended.

R3-A1 envelope SHA: `b45a048923102feaa2040a769281cc2c258d66be09c2ce3561238b4b74c3f932`;
Coordinator snapshot SHA: `29595284a6e4202129d9f014aee6e1fec545d19bdfe028411687d41ec6b8fa17`.
Initial grant, slot activation and amended snapshots remain separate.
Overseer assigned Scientist the exact R3 slot at `2026-10-08T00:20:56Z`, after
Worker released it at `00:16:55.5233253Z`. Pre-native amendment `r3-a1` at
`00:30:59Z` clarified contiguous Government then Classic blocks, each with its
own positive entry checkpoint, without changing any quota. Actual starts
revalidated the exact active live status, Scientist slot, grant key and timestamp.

Government remains pinned to `04e225d5caee78c2a198607143863fca1e829750`;
Classic held source remains `c91363b7ac4decbe87212ff0f588b5451581a152`, Runtime
`7dbd599c81540c8203a1b7f83afbc335174f4f1f`. No P1 candidate was substituted.

## Actual outcomes

Government: one Queue, native OS return 0, native job/queue `incomplete`.
One Executor wrapper and one deterministic delegate returned protocol-valid
bytes, then the held native host rejected `candidateJson:null` as a nonempty
`json.RawMessage`: `executor response contains verifier or inference output`.
The missing evidence identity in the failed native run report is a separate,
downstream normalization qualification. The product stopped at this first
negative checkpoint; no Resume, repair, retry or extra native call followed.
Read-only attribution is in `evidence/native-integration/run-3/government/failure-attribution.md`.

Classic: all five native calls completed the declared mechanical flow:
Execute `planned`, exact independent external review, Apply
`materialized-unverified`, fresh Verify `passed`, Audit `complete` with no
findings/next steps, then stale Apply `refused` with no second writes. The
stale replay returned 2 as the expected refusal. Classic controller elapsed
92.09 seconds within its 180-second window. Three fixed deterministic role
slots ran: Executor Dotnet, Verifier Dotnet and Verifier Markdown.

The structured external Execute review binds raw report SHA
`4b746df3082120c05f7d0f2310ade6e37e6dbd2c950c4ccf2b14fb59ebcabf9f`
and exact digest; approval SHA is
`cbad08e1bf6038ef21224194190720137b140a809db49a6bb9e8228e80874ee3`.
Its original human-readable narrative had shell-rendering errors and remains
unchanged; a separate explicitly post-run erratum supplies the readable
values/rationale. That erratum creates no new or retrospective execution authority.

## Additive accounting and evidence

| Boundary | Native starts | Wrapper attempts | Deterministic delegates | Reserved seconds |
| --- | ---: | ---: | ---: | ---: |
| Historical R1/R2 | 5 | 3 | 0 | 750 |
| New R3 Government | 1 | 1 | 1 | 150 |
| New R3 Classic | 5 | 3 | 3 | 750 |
| Cumulative fixture history | 11 | 7 | 4 | 1650 |

The unused Government Resume reservation is not available for another case.
The original native-start database was appended, preserving all five prior row
tuples and the R2 correction record. Historical role-ledger bytes remain
unchanged; the historical Government controller remains in its original state.
Government's actual outer product receipt is in `controller_runs` (zero
`controller_processes` rows); Classic has five action rows. Count the Government
outer receipt once and each Classic action once.

`quota-and-ledger-summary.json` and `external-snapshot-manifest.json` retain the
closed ledgers, raw product/wrapper/delegate receipts and 2303 verified original/
copy pairs. The archive includes the fresh Classic fixture source tree. The
150 seconds per native start is a logical reservation including role/cleanup
allowance, not measured aggregate OS descendant time. Actual process times are
retained separately; every native process stayed within the 38-second bound.

No new real study Actor, model/provider run, metadata session or study cell was
started. Historical known reported tokens remain 53331; their true total remains
null. Unknown native/provider usage fields remain unknown. Explicit fixture
zero-provider accounting does not convert those unknown fields to zero.

Focused offline gates covered 38 selected tests across retained logs/checkpoints,
including the real unmocked Grant/Classic bind_request/bootstrap chain. The
initial wrong-module command error is retained alongside its corrected module
run; amendment and checkpoint changes received targeted reruns. These are
harness checks. No full product suite, product source change, push, PR, release
or publication occurred. Independent preflight and post-run reviews are under
`public/native-integration-r3-*-review.md`; delivery hashes are in the new
`evidence/native-integration/run-3/delivery-validation.json`.

Classic's result establishes bounded native deterministic integration mechanics.
Government's result identifies a deterministic response-shape incompatibility.
Neither establishes S1, real Actor capability, semantic quality, verifier
independence in a study, economic benefit or human acceptance. No historical
FAIL/INVALID/BLOCKED/NOT RUN result is rescored. Further source work or actual
runs require the next explicit bounded Overseer assignment.

## Accepted R3 offline package

Completed package: `s1-offline-adapter-contract-repair-r3-preparation`, delegated
by Overseer after accepted r2 delivery
`419775ab0ea5f421655ab3c8bd97e286ffbb6e0a`. The local commit containing this
handoff owns the R3 source/evidence snapshot; exact file hashes and historical
preservation checks are in
`evidence/native-integration/run-3-preparation/delivery-validation.json`.

The Government/shared delegate contract now explicitly binds the executable
to `argv[0]`, preserving digest/runtime-file checks and the launch guard.
Classic now resolves native Projection/Definition identities and matching
context/authorization, with all three slots covered by static preflight.
The focused gate passes 18 tests. The six-role chain exercises real temporary
Authority/bootstrap and per-Invocation resolver/reservation code, then mocked
success/failure translation for each role. Source-grant admission and Classic
product binding remain patched boundaries; no native or delegate execution is
proved. Original failures are reproduced offline from frozen/source-bound
inputs, without reconstructing missing full historical stdin.

Independent review: `public/native-integration-r3-offline-review.md`, SHA-256
`f638cc31105b155a09a15b324f8e5f1079026ac9930e0b214b9377407a0914e2`;
no material finding remains in the assigned offline scope. Full report:
`public/native-integration-r3-offline-preparation.md`. New R2 errata:
`public/native-integration-r2-government-errata.md`; the cited final native
ledger has five rows, and Government's outer process receipt is present.

**STOP: no new run quota, native product/controller/queue, real delegate,
model, provider, metadata or study start.** Both R2 cases remain incomplete;
S1, native lifecycle success, semantic quality and human acceptance remain
open. Product pins and historical ledgers/freezes/reviews stay unchanged.

`evidence/native-integration/run-3-preparation/next-native-allocation-proposal.json`
is proposed only: first checkpoint Government queue + Classic Execute requires
two newly granted starts/300 reserved seconds. Full conditional lifecycle
requests Government +2/+6 wrappers/+6 fixed delegates/300 seconds and Classic
+5/+6/+6/750 seconds, with zero real Actors/providers/metadata/study cells.
A new exact Coordinator-bound finite grant, narrow reviewed admission update,
fresh disposable bindings and immutable preflight freeze are prerequisites.
Do not reuse closed R2 authority, refresh old runtime snapshots, repair/relaunch
a failed native case, or resume execution after compaction without that grant.

## Closed r2 record

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
