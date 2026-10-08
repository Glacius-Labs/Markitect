# Independent review: Government check-receipt identity

**Disposition: the narrow receipt-identity correction matches the pinned Government host contract and closes the observed pre-Resume mismatch without widening acceptance. This is offline source/test review only; I did not run tests, admission, preparation, freeze, SQLite writes, or a process.**

At pinned product commit `04e225d5caee78c2a198607143863fca1e829750`, a configured `authoring.Check` has `Name`, `Run []string`, and optional `TimeoutSeconds`. `planVerifyCommands` validates the check, maps `Tool` to `Run[0]`, and passes `Run[1:]` as arguments. `host.GateResult` contains exported `Name`, `Tool`, `ExitCode`, `Milliseconds`, `TimeoutMilliseconds`, and optional `Output`; its fields have no JSON tags, so Go `encoding/json` emits those exported field names. The report exposes the results under its `checks` field. These are the pinned producer semantics, not inferred from the R6 result alone. The frozen `inventory-overflow` configuration uses `run: [go, test, ./inventory, -count=1]`, and its archived receipt correctly reports `Name=inventory-overflow`, `Tool=go`, exit 0, 6,270 ms, and a 30,000 ms timeout.

The correction derives the expected identity as exactly `(check.name, check.run[0])`. It requires a nonempty command list of strings without NULs and a first token that is a bare executable name. It rejects malformed observed identity fields. The existing exact-set and count comparisons still reject missing, extra, duplicated, or changed `(Name, Tool)` entries. The existing strict numeric checks remain: integer exit code exactly zero, integer nonnegative elapsed milliseconds, positive integer timeout, and elapsed time no greater than timeout. There is no fallback to `tool`, no command-string parsing, and no additional executable/argument grammar.

The neighboring R6 pre-Resume consumer is `FixtureBudget._profile_validate_queue_success()`: it binds the successful process receipt and exact Request, invokes the real `government.translate_queue_result`, loads the digest-bound runtime check definitions and report checks, then calls the corrected helper before the subsequent Resume reservation/start path. The new offline regression uses sealed R6 request/runtime/report bytes, relocates them into temporary copies, and calls this real consumer/translator path; it checks acceptance of the exact receipt, rejection of a changed tool, and retention of runtime/report digest guards. The tests forbid `subprocess.Popen` and `sqlite3.connect`. These mechanics do not show that Resume ran or that an end-to-end native retry is authorized.

The retained focused log reports 13/13 passing tests (12 new offline regressions and the directly affected existing R4 helper test); I read the log and did not rerun it. Offline preservation validation reports 3,711 protected files unchanged, all 92 prior R6 raw-archive pairs unchanged, the live ledger unchanged at SHA-256 `c0a91051a16f5d2d3a7754789c2ad02b67b6b2f57f32ac4bd16562700b8fcf1a`, and zero new preparations, admissions, freezes, native starts, controller starts, wrapper starts, delegates, or live-ledger writes. The earlier R6 freeze is explicitly invalidated because this authorized consumer source changed; it was not reused. Provider/token totals remain unknown.

Source and evidence bindings reviewed:

- Pinned host `verify.go` at `04e225d5caee78c2a198607143863fca1e829750`, blob `f00566e6935e8556cac6d5daa3c1eb7c2eb79ec0`, SHA-256 `e8beb39c8706daffa8fac0fcd4d6d1d5defb17ce3675bae26be736bb3ed3eb0f`.
- Pinned host `authoring/check.go` at the same commit, blob `b42bbf9177c151005fc642428202f6202115a2fa`, SHA-256 `8bcb2a08ebb122fe4012e75f61d580fa2d06d922196c3b484aec3292a835c68d`.
- `runtime/native_fixture_budget.py`: SHA-256 `a5c95a9bd4ad7cdeaa7d1cfec13c738835021082767e9f5f7a19ddd5ecd3a3e5`.
- `runtime/test_native_r4_budget.py`: SHA-256 `22567767ec4d1ca68555c89cd5f3936e204ee50975d33108126d52334f12e914`.
- `runtime/test_government_check_receipts.py`: SHA-256 `00b72b01937ba817a24105cfb2cfda451abcc9468e9d6c60a9a478d910ac5f78`.
- Focused test log: SHA-256 `4cd084af3497d67fc0eebc369d4e5eae8dbf56dc52284756d626fd3afc492c66`.

The R6 run remains terminally incomplete; this offline fix does not retroactively authorize or establish conditional Resume, S1, semantic acceptance, product quality, or human acceptance.
