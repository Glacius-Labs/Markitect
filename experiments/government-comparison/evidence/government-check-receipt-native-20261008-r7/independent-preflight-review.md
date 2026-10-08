# R7 independent source preflight review

**Disposition: no material preflight blocker found in the R7 closed-profile, budget/receipt, driver, or transition-test changes reviewed. This is source and retained-offline-evidence review only. I did not run tests, preparation, actual admission, freeze, ledger writes, or processes.**

The R7 profile is a separate identity: key `government-check-receipt-native-20261008-r7`, dispatch `government-native-check-receipt-r7`, task `positive-overflow-release-r7`, Request marker `nativeFixtureR7Grant`, and Protocol/Grant metadata `fixtureR7SourceGrant`. The exact Scientist grant pointer/thread, envelope, coordinator snapshot, base commit `18c9f907863044972511fb2403bcdb4eced95810`, and assigned slot/time `2026-10-08T04:10:48Z` match the profile pins. Envelope SHA-256 is `639af044fc46b5a996b8a1d80f44cbe3d8861257b44286bcef2a6a6cb6305331`; snapshot SHA-256 is `d17258c72555a1d1a3da3e962f9542fe964b34edbab80950b918d3e379635078`. The embedded Scientist grant matches the envelope grant, ignoring JSON object key ordering.

R7 carries the authorized one-fresh-case/one-explicit-admission limit, at most two native starts, six wrapper attempts, six deterministic delegates, parallelism two, 150 seconds per reserved start and 300 new seconds total. Its cumulative ceilings are 15 native starts, 19 wrapper attempts, 16 delegates, and 2,250 reserved seconds. The profile binds the immutable 13-start/4-correction R6 terminal ledger (SHA-256 `c0a91051a16f5d2d3a7754789c2ad02b67b6b2f57f32ac4bd16562700b8fcf1a`) as its starting history. R7 prior labels include the existing R6 Queue row, but not an R6 Resume; the R7 correction identity is separate. The R5/R6 keys, dispatches, markers, paths, quota identities, and existing histories remain distinct. Mixed profile markers and wrong dispatch/task combinations fail closed.

The shared driver requires `host-success-contract.md` for R6 and R7; the R6-only a1 addendum remains R6-only. It binds the R7 offline tests in admission and freeze evidence. R7 uses the existing one-shot actual-admission claim/receipt flow, while mandatory side-effect-free Authority revalidation of the same frozen Request remains required before Queue and any conditional Resume. Failure closes the case without repair/retry. Native and controller deadline gates remain 38 seconds, with the existing 32-second work window and cleanup reserve. Resume remains conditional on the complete native and outer Queue proof, strict fresh check identity/outcome/timing, and exact Queue/result/receipt binding. No old profile is reactivated or relabeled.

The corrected fresh-check consumer now derives the expected tool only from `valid Check.run[0]`; the pinned host producer emits that executable in `GateResult.Tool`. The R7 transition regression uses the real `fixture_grant_bindings` producer, strict R7 grant validator, corrected pre-Resume consumer, and real queue translator on temporary copies of the sealed R6 case. It rejects a changed native tool and retains runtime/report digest checks. `Popen` and `sqlite3.connect` are forbidden in the test fixture. This tests adapter/consumer mechanics; it does not simulate or establish a successful native R7 case.

I inspected the retained final offline logs without rerunning them: 18 shared R5/R6/R7 budget tests, 7 R7 chain tests, 13 sealed binding/checkpoint tests, 7 R7 checkpoint tests, and 3 real-consumer receipt-transition tests passed. The offline validation records zero R7 preparation, admission, freeze, native/controller/wrapper/delegate starts, provider/metadata starts, study cells, or live-ledger writes; prior historical files and R6 archive pairs remain preserved. No actual-input validation has occurred. No product-quality, S1, semantic, tool-capability, provider-usage, or human-acceptance claim follows from these offline results; the six base study cells remain unrun.

Source/evidence pins reviewed:

- `government_native_profile.py`: `399a568e0ccab6781b3ecff8d4a7cbe479b530b69e5e3239711d5cdcf94452a0`
- `prepare-native-r5.py`: `09966abe55947d89cdf18817082da976af12c0d65eb0431dc437ba16589feeb0`
- `run-native-integration-r5.py`: `fa2b8d6565db65c84af0bfabb697850e29bdd7162e61346696327edc02b0cb74`
- `native_fixture_budget.py`: `1e9c05326759aa86a4cbb3498bde20b7aed0fce80f6c77eff17b1776ab2d4bd1`
- `native_controller.py`: `56fee7067ff606a5983456b0598eba01c3c073690dedeab1e7d845b21f8270d2`
- `government_roles.py`: `743e1e0f3671f81913ca35e2b232662b9b51eaeba9783370d946fc42027dec90`
- `dispatch.py`: `7b78d26e430ea8474ac1e6d7af286e965946a115c9328e828f8483c74da5f296`
- R7 budget, chain, receipt-transition, and checkpoint tests: `a2f09d49b94baf546bbad8c68a79b559a00aabfb23f738a338d2d0e13da9d77a`, `b46df09b953fede91918707fa40b8b1bc02c0c5357cb2a671d75800703163ce0`, `b647f23fabc778b09c8c5874eaddb9f6f434c59484f55c715b700172734681cd` (receipt-transition), and `9586295cd8738100001c6c4ef890547a5c5120af9c677d76a4895699b071a9a8`.

No actual R7 execution is authorized or claimed by this review. The prior R6 run remains terminally incomplete, R5 remains NOT ADMITTED, and R4 remains incomplete.
