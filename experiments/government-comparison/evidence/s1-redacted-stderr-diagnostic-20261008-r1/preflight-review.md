# Independent source and preparation review

Reviewed the candidate source and captured offline binding-test results read-only. This is not an execution or runtime preflight: I did not run the client, tests, candidate executable, or any process capable of launching Codex. No request, freeze, reservation, or external evidence root existed in the review snapshot. The canonical slot was assigned to this exact Scientist key at `2026-10-08T16:52:42Z`; the grant remains issued at `2026-10-08T16:47:18Z`.

The copied authorization grant semantically equals the current canonical `Scientist.evidence.redactedStderrDiagnosticGrant`, including JSON value types. The active slot owner, key, status, `grantIssuedUtc`, and later `assignedUtc` match the grant's required activation fields. The copied whole-coordination hash is provenance only; unrelated cursor/time changes are not authority drift. The snapshot showed no request/freeze/reservation and no external output root, so those exact bindings and absence conditions still need to be established by the root coordinator before any one-shot entrypoint use.

The new client is a narrow derivative of the accepted offline consumer wiring and R2 outer. It retains the five-method allowlist and existing protocol/thread gates. The profile keeps the 17 configuration pairs, exact pinned binary and schema archive, two expected public input files, read-only/never/ephemeral thread payload, and prior model/provider/reasoning fields. The new capture binding consistently sets input to 1024 bytes, at most four complete physical lines, redacted output to 2048 bytes, and `private-redacted-stderr/redacted-stderr.json`; the diagnostic contract pins the previously accepted collector. The complete live grant, exact active slot and binding identities are checked in the source. Tests cover semantic grant equality and JSON types, slot ownership and later assignment, unrelated state cursor/time changes, request/freeze identity/order, and the capture/path/collector pins.

`run_once.main()` performs the full binding check before creating the absent external root, then repeats it immediately before writing the durable reservation and requires the profile, request, freeze, source head, and slot assignment to match the first check. Root creation is exclusive; unexpected existing content stops before reservation. The controller validates the binding before worker creation, again after Windows Job assignment and before sending `GO`, and the gated worker validates it again after `GO` before starting the app-server. The source review found no remaining material authority or lifecycle gap in these boundaries. This is a sequence of checks, not an atomic lock over the separately owned coordination file.

The session uses the same `stderr_seen` event for the pump, collector, and RPC admission. A nonempty stderr read sets the stop event before acquiring the lock, so a later RPC cannot be admitted after observed stderr; an RPC write already admitted may still be in flight. Continuation reads are cleanup-only. Private excerpt creation is one-shot and allowed only after cleanup has started, the native process has exited, all pumps are quiescent, and the deadline remains open. Receipts contain metadata only. The collector's caps and path are checked against the grant, profile, contract, and pinned collector. No excerpt text or content fingerprint is included in the public result.

The targeted synthetic binding suite recorded 6/6 passing in 0.017 seconds (0.237 seconds wall time), exit code 0. These tests do not establish native CLI/schema compatibility, app-server behavior, successful redaction of unknown secret forms, OS/ACL isolation, provider usage/billing, or task-history deletion. A permitted sanitized read can persist in task/model history. Partial-output status does not alone prove file ownership against a same-user insertion race; only a separately proven exclusive file creation permits removing a partial file. Do not read a partial artifact.

Before source commit, the new packet's copied text/JSON files were normalized to LF to match repository Git settings. The frozen method-enum JSON and schema-contract JSON remain semantically deep-equal to the preserved R2 copies; the new client changes its `METHOD_ENUMS_SHA` constant to the LF-byte hash of its unchanged enum JSON. The focused test result is unchanged; its log was normalized to LF as well. Historical files were not modified.

Current grant bounds are one diagnostic reservation, at most one new app-server tree (seven prior, cumulative maximum eight), zero Actor-task reservations, one thread start, zero turns/tools/CLI metadata calls, parallelism one, and no retries. These are prospective limits, not evidence of any reservation or execution. A validated thread response is terminal before any turn. Historical usage and billing remain unknown beyond the existing recorded counters; no root cause, product, general S1, or acceptance claim follows from this preparation review.

Reviewed file SHA-256 values:

| File | SHA-256 |
|---|---|
| `client.py` | `70bf48d517274eb95df2741d737cc3efbbde38938ce1d23ff1062a5f55f4832b` |
| `run_once.py` | `14fa7ecf2c75cd0bdf212f19cf4d16a8ce8d52b57a59e88ecf628f2418861619` |
| `test_bindings.py` | `65adbbd26ab14d711ec5709fee5ec5d51381a0f159a6bed03e0a5ea19563d956` |
| `focused-tests.log` | `d0db283f665aa60b717011439ea92cf9e1d93c3ec0a87f785443ecc061dfc80a` |
| `authorization-grant.json` | `894ae2c792f7840fcd0ef7fc69d6a2da1d6ad1e77e4778992cdab30987dd1905` |
| `profile.json` | `aed4b6f57326a664342f5d989493117551715337a0e8d8fc666d3c5e7d8e6b6e` |
| `diagnostic-contract.json` | `b01f58965acc80908c7ed52c81b4fa9abe73eb8a73f213272dff2c6c90750771` |
| `schema-contract.json` | `c4bf557d17426f4c9a6c61e97353b8361469169e067e1f911fdfc7488d369270` |
| `frozen-method-enums.json` | `831728c4ce8c1913d25eff43229c93b81c0f5d802f93eb3b057333c4048e2220` |
| `thread_gate.py` | `b48ce8b67e3a18ecab80a49d1eb3633f90443cc9524972e00a3ea7bdf27fa6cd` |

The accepted offline consumer source remains pinned at `ac665dc546e7e333fcaf0e97e011e22b606d8809df2e47d23fdddb351d3e1dc8`; its accepted collector remains pinned at `d92f79b0d14abd2c5851bea2162813c5f93b396e4d19fc927389bc8b6557ef6d`.
