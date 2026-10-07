# Native integration mechanics (S1)

This supplement describes the current reusable harness boundary. The previously
accepted [`s1-adapter-mechanics.md`](s1-adapter-mechanics.md) remains unchanged
and SHA-bound. These additions do not authorize a study, Actor, or provider run.

## Acyclic controller bootstrap

`runtime/native_controller.py` exposes `write_bundle(...)` and `load_context(...)`.
The coordinator validates and claims a nullable-Actor controller row in the
common Ledger, then writes an immutable bootstrap bundle binding the exact
Request bytes/path, validated Authority Grant and Protocol, released role
authorization, captured input bytes, product source/binary pin, and controller
claim. The launched wrapper receives only an explicit bundle path plus its
SHA-256 in the environment. The helper rejects a missing or changed bundle,
unclaimed controller, or mismatched paths/digests. It strips the locator
variables before starting the configured role delegate. This is an
authorization/accounting boundary, not an OS isolation claim.

The mechanical fixture source grant is supplied on the outer Request as
`nativeFixtureGrant: {path, sha256, sourceKey}` and the identical `{path,
sha256}` must occur in `releasedInputs`. The grant is read from the bound source
snapshot and constrains product pins, zero real Actor/provider/study use, and
the finite fixture allocation. The no-provider grant is rejected unless
`Request.mode` is `mechanical`. Dynamic Request/Grant/Protocol digests are kept
in the outer bundle; they are not pinned as static RunnerSpec files, avoiding a
digest cycle.

## Role bridge

`runtime/government_roles.py` validates the native stdin JSON Invocation and
exact response identity echo, maps the configured role slot to its authorized
delegate, checks path and runtime-file digests, computes a deadline-bounded
timeout, and reserves one unique invocation in the common Ledger before
starting the delegate. A single transaction checks Coordinator session/token
limits and, when the source grant is present, its per-product deterministic role
start and reserved process-second ceilings. Completed process time replaces the
reservation; ambiguous calls remain consumed. Unknown usage remains unknown.
This preserves the native `sha256:` input digest as an opaque identity and does
not attempt to recreate Go request canonicalization.

The outer controller has its own elapsed-time and process-receipt rows and
consumes no Actor attempt. Every delegate role call has a normal common-Ledger
Actor reservation. Government's dispatch path requires the exact source grant
and parent `FixtureBudget` claim before a native queue process; queue resume is
not automatically retried.

## Classic controller API

The reusable API is:

```python
session = begin_classic_controller(request_path, authority,
                                   request_raw=request_bytes,
                                   captured=captured_inputs)
run_classic_step(session, "execute", fixture_budget=budget)
run_classic_step(session, "apply", fixture_budget=budget,
                 external_review=review_bound_to_execute_bytes)
run_classic_step(session, "verify", fixture_budget=budget)
run_classic_step(session, "audit", fixture_budget=budget)
run_classic_step(session, "apply-replay", fixture_budget=budget,
                 external_review=the_same_review)
result = finalize_classic_controller(session)
```

The Request binds the exact finite action sequence. Execute produces the report
that the external review must identify by raw-byte SHA-256 and native digest;
Apply is planned only after that review is supplied. Verify must use the same
controller's materialized-unverified Apply result. Audit and stale-Apply replay
are separate recorded processes. Each start requires the parent `FixtureBudget`
for the exact grant and pinned binary and stores its immutable process receipt.
The normal `dispatch()` path remains closed for Classic stateful work.

## Verification boundary

The captured focused Python gate passed **88 tests**. After final static
API-version and RuntimeFiles-deduplication changes, the affected gate passed
**34 tests**; the final fixed-delegate accounting/gate checks passed **23 tests**.
Receipts are `evidence/native-integration/run-1/focused-tests.log`,
`final-delta-tests.log` and `final-fixture-accounting-tests.log`. These gates
validate mechanics, not a successful native workflow. The translator checks
configured role receipts and positive native decision/vote bindings; it does
not infer independent human identity from native slot labels.

The parent subsequently made three authorized native starts: Government inspect,
Government queue and Classic Execute. Both positive cases ended incomplete;
one Government wrapper was attempted and no deterministic role delegate started.
The [negative handoff](../native-integration-handoff.md) records the exact pins,
freezes, missing wrapper stderr, unfinished Government controller bookkeeping
and Classic RecordStore setup failure. No resume/replay, provider, study or real
Actor start followed. The fixed-delegate accounting path was not reached by
either native execution; S1 remains open.

The subsequent r2 correction adds a separately source-bound additive grant,
preserves the original native-start ledger, caps early wrapper invocations at
six per product, retains raw wrapper stdout/stderr before dependent imports,
requires an absent Classic RecordStore root, and terminalizes new failed
controllers with their validated receipts. Its focused gates passed 102 tests
and then 9 final authority/controller tests. The two r2 native calls also ended
incomplete, each before delegate reservation. Both new controllers ended
incomplete; the historical r1 Government running row remains unchanged. See the
current handoff for exact source/freeze bindings and cumulative consumption.
