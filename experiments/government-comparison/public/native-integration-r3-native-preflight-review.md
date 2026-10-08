# R3 native integration preflight review

Date: 2026-10-08

I independently reviewed the frozen R3 runtime candidate `36ac50558eabcb44812b2c85d723c9da5b783e4c`, the subsequent driver checkpoint fix `0f24deb6534075922acdf35ccf8bfcd330cbf24a`, and the preflight evidence delivered in `94ccbc2fe5efc0f1655bb73695a9c22da16659ec`. I found no remaining preflight blocker in the reviewed grant binding, budget admission, product request binding, or controller/delegate gates.

The final driver SHA-256 is `5fb4b08a321c06fc744503567238b303470a4b927c1cdc732f5fe400f6ad265b`. It gates every positive Classic continuation on a digest-matched process receipt, the original parent-captured stdout digest, return code zero, no stop/timeout, and the exact expected report status; Audit must also have no findings or next steps. The two offline checkpoint regressions pass, including tampered stdout, changed receipt, timeout, stop, nonzero exit, wrong status, and nonempty Audit results.

## Bound source and authorization

The R3-A1 amendment keeps the existing R3 grant key and limits, and resolves sequencing as contiguous Government work followed by contiguous Classic work. There is no overlap between product controllers; the two entry starts are separate checkpoints, not a joint temporal barrier. The grant permits at most 2 new Government starts / 300 reserved seconds and 5 new Classic starts / 750 reserved seconds, within 7 new starts / 1050 seconds and 12 cumulative starts / 1800 seconds. The first error or unmet prerequisite stops that product without repair or retry.

| Input | SHA-256 |
| --- | --- |
| Original R1 source grant | `b917f5a5eb99f0a607fd282a81acdf7b085e6a1dba14f89c8d0c32f349c82c3e` |
| R3-A1 grant envelope | `b45a048923102feaa2040a769281cc2c258d66be09c2ce3561238b4b74c3f932` |
| R3-A1 Coordinator snapshot | `29595284a6e4202129d9f014aee6e1fec545d19bdfe028411687d41ec6b8fa17` |
| Immutable five-row native-start history | `c5769c0c204cfbc5d17d95320c8b2288b746aa785c16796ce0d7ab433b58fbf5` |
| Government Request | `d36c48f005bd5721529900588b21ded8b0ed1234db515c0bc23d445f70085794` |
| Classic Request | `e12f9a0ca39f770cfbb9842858208dd8373d3986459406c323d2995bc3c9d1de` |
| Source checkpoint | `6f2d664bc72cfe765cd67e561e89cf1423b8971154645072101c545ee2f6b18c` |

The frozen request inputs under `evidence/native-integration/run-3/preflight-inputs/` match the external released Government and Classic inputs byte-for-byte: 12 Government files and 11 Classic files, with zero hash mismatches. The fresh actor trees were recorded clean at `c852336817c156027441c9f12aa737359a5e6613` (Government) and `ba061721da875f0201e2c0c23a795b2ec300a78c` (Classic). The relevant source pins are the Government producer at `04e225d5caee78c2a198607143863fca1e829750` and Classic held source `c91363b7ac4decbe87212ff0f588b5451581a152`, with the exact accepted binaries recorded in the R3 grant.

The live Coordinator state at review time matched the exact active R3 status, Scientist owner, R3 key, and slot assignment timestamp. `validate_r3_entry_gate` now requires that exact active status; its regression test rejects `Closed`, `revoked`, and unrecognized status strings, and also rejects mismatched slot keys or timestamps. The R3 budget validator requires the original R1 allocation and exact five historical rows, permits only R3 labels, preserves cumulative accounting, and enforces Government-before-Classic order and one active native controller at a time.

## Offline checks and limits

I independently ran the focused offline tests: `test_native_fixture_budget.py` passed 9/9, `test_r3_classic_preflight.py` passed 1/1, and final driver `test_native_r3_checkpoints.py` passed 2/2. The Classic test exercised real Authority validation, Classic `bind_request`, R3 grant admission, role preflight, controller bootstrap, and `load_context` in a disposable temporary case; it rejected missing R3 binding, mixed R2/R3 binding, and a changed Classic scope before ledger claim. It finalized as incomplete with zero product processes. The persisted source checkpoint separately records the selected preparation gates and retains the initial command error in its own log.

Both before-run budget receipts show the same five finished historical rows, no new row, no refill, no provider calls, no real Actor start, and no study cell. The original native-start database and archived copy both hash to the historical value above. No native binary, product process, wrapper, deterministic delegate, model/provider session, metadata session, or study cell was started for this review.

This clears the reviewed offline binding and accounting preflight only. It does not establish native runtime behavior, semantic quality, S1 completion, or human acceptance. Any later native start still requires the parent’s immutable final freeze, a clean source/evidence state, and the live grant gate immediately before reservation and effect.
