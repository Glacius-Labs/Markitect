# S1 infrastructure checkpoint — 2026-10-07

Source candidate: `4f615a0e5866c0e045d6dd2371678622a02d9cdf` on
`codex/government-scientist`. This records finite infrastructure mechanics,
not autonomous study performance or human acceptance. There were **zero native
Actor/model/inference starts and zero study cells** in this package. Mechanical
fixture processes use separate synthetic test ledgers; their counters do not
describe a real provider.

The exact-source final command was:

```text
python runtime/verify_s1.py --evidence ABSOLUTE_PATH_TO_THIS_NEW_DIRECTORY
```

It passed **45/45** focused tests in 10.394 seconds: 25 dispatcher tests,
12 synthetic context/access tests and 8 existing ledger/process runtime risk
tests. `targeted-tests.log` is the raw report; `observation.json` binds source,
Python executable/version/hash, unchanged historical ledger hashes and outcomes.
The source compiled via `py_compile`. No Go suite, Go Check/Context/Verify,
Brownfield fixture suite, six-cell preparation or native product execution ran.
Earlier source gates remain historical; no new full source-gate claim is made.

`mechanical-records.zip` preserves 241 generated records from the external
deterministic workspaces and the artificial context package. Its SHA-256 is
`8f79bb512b8295f33ba8612267eaa11abb28e377ff688e51c718711deae67762`.
`mechanical-records-manifest.json` lists each archived path, exact byte count and
digest. These are local, cooperative hash-bound records, not tamper-proof
attestations. They include intentionally incomplete/blocked/unknown recovery
fixtures and invalid authority fixtures used to test refusal.

## Verified mechanics

- Public harness API and actual adapter CLI use the same explicit operator grant,
  protocol, exact Request and execution binding. Mode alone cannot launch native
  work, and arbitrary mechanical executable/argv routing is rejected.
- An atomic shared reservation precedes the process claim; concurrent booking
  respects four active sessions and duplicate dispatch starts exactly once.
- A completed deterministic process stores raw usage and a durable Result; losing
  the derived Result file or resuming a terminal receipt does not relaunch or
  charge counters twice.
- Shared stop/deadline kills owned local work and the synthetic descendant; the
  descendant's delayed marker is absent. This says nothing about remote provider
  work beyond the owned process tree.
- Missing usage remains null and blocks admission. Observed counters cannot
  decrease. Synthetic parallel subtotals demonstrate retrospective overshoot,
  rather than a hard in-flight token cap.
- Interrupted booking/recovery closes consumed without launch. A launching
  record without a terminal receipt stays active/blocked without refill.
- Artificial context/access preparation binds exact prompt/card/helper bytes,
  read/write/process expectations and checked observations. The same-user helper
  can read the sibling control and run its known harmless child. Actual Actor
  context, rights and filesystem isolation remain untested.

## Independent review

The independent reviewer `/root/independent_review` reviewed dispatcher,
accounting, public CLI, verification/freeze helpers and the profile proposal,
and independently ran **25/25 dispatcher tests**. No material findings remained
after re-review. Findings corrected before this source candidate were:

- unconstrained Result paths → dedicated grant-bound external output directory;
- listing assertions without content checks → exact successful authenticated
  metadata content validation;
- prelaunch inference claims → false before launch and unknown for native
  ambiguous/observed execution;
- missing Unified Exec disclosure/reactive tool classification → explicit gap,
  polling/finalization classification and contamination flag;
- lossy usage observations → monotonic counters and regression refusal;
- prompt reread/mutable release paths → captured prompt bytes, separated releases
  and retained/verified snapshots;
- two-transaction unlaunched recovery → atomic consumed terminalization, including
  old recovering state.

Parent review additionally corrected the access card's newline/hash mismatch
and restricted the synthetic helper to the exact current Python and generated
source template. The 12 context/access tests include consistent-manifest tamper
refusal. Receipt consistency is still not an OS privacy boundary.

## Unchanged authority and remaining work

The original resource proposal is byte-for-byte unchanged from `b16d704`.
The exhausted historical two-start ledger remains SHA-256
`74dfd51e2802bc67131891b28c6858129ff5ef4a50a6f1fe06a8793a7250732f`;
the exhausted selected-runner diagnostic ledger remains
`33bd2ba59b11ab6d54f75c55e6d12b00ec6b90df00106a1419aed239cef2982c`.
No fourth diagnosis or valid live Run Grant was issued.

Native provider-request/internal-retry counts remain unknown; agent turns never
replace them. The current strict rule therefore ends such a native smoke
incomplete and blocks subsequent admission even if tokens are known. The
[common measurement-profile correction](../../../public/s1-profile-correction-proposal.md)
is proposed only and requires explicit adoption plus a reviewed cumulative-ledger
follow-up before a multi-session sequence. No reset/new-trial workaround exists.

The [next resource proposal](../../../public/s1-next-request.json) requests exactly
one later artificial context session: ≤180 seconds, parallel one, no wrapper
retry/continuation/subagent/repair, retrospective 10000-token threshold, then hold.
It still needs a valid Coordinator Grant and frozen exact public Request/protocol.
Separate later access and product smokes have no implied authorization.

Classic remains v0.14.1, runtime 7dbd599 / held c91363b. Architect 8927704 is not
promoted to study candidate. Coordinator-accepted Government G2 at 2b604d1 was
read as a public intermediate native interface; no Worker code or binary was
changed/run here. It is not a final StudyGovernmentVersion freeze, G3–G5 proof
or real model quality evidence. Both native product adapter integrations still
return readiness_gap before reservation. **S1 remains open.**

The r3 chronology correction is prose only: targeted-tests-final.log contained
five tests; targeted-tests-gate-fix.log contained six after the gate fix. Neither
historical log was changed or rerun.
