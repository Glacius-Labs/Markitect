# Independent postreview: preparation blocked before reservation

## Review conclusion

The terminal classification is supported: grant `s1-thread-start-diagnostic-20261008-r1` ended **PREPARATION BLOCKED BEFORE RESERVATION** because the exact required executable was absent at `C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/5ea220ae823df3d7/codex.exe`. A direct read-only existence check returned false. The packet records that final request/freeze generation attempted to read that exact path before writing either file and received `FileNotFoundError`. The cause of the absence is unknown. No alternate executable was searched for or used.

I found no evidence of a diagnostic invocation or reservation. The new external evidence directory exists with zero entries. `request.json`, `freeze.json`, `reservation.json`, controller markers/receipts, `started-once.json`, `sanitized-result.json`, and `outer-receipt.json` are absent. This agrees with the terminal summary and slot-release records. No tests, binary, client, controller, Codex process, or native diagnostic was invoked in this review.

## Independent read-only checks

- Source commit `S` resolves to `7e9271f0dbc81d202c92cf627999723cc750c6ad`, based on `befbd24d625c77f2f0ab1c36bb91e36b65473e9e`. I compared each of the 17 entries in `preservation-validation.json`'s `newSourceFilesAtS` manifest with both the current working-file Git object (`git hash-object`) and the blob at `S` (`git rev-parse S:path`); all 17 match the recorded hashes and each other.
- Direct decoded-JSON comparison confirms the copied authorization grant equals the current `Scientist.evidence.threadStartDiagnosticGrant`, and the copied slot object equals the current canonical `fullSuiteSlot`. `authority-binding.json` retains the earlier and current whole-file coordination digests separately. The older digest is not represented as current.
- `preservation-validation.json` records 4,029 historical package files and working bytes unchanged, five live ledgers, prior request-source pin counts of 15/21/27, 21 external original/copy pairs, and the relevant pinned schema members. It marks the mandatory executable as the sole unavailable item in each of the three prior freeze-input sets and preserves their historical records. This does not retroactively verify current binary availability for those old frozen runs.
- `terminal-summary.json` and `slot-release.json` consistently record zero new diagnostic/Actor reservations, zero new app-server trees, zero thread/turn/tool requests, no retries, and cumulative totals unchanged at six app-server trees and six Actor reservations. They preserve five historical model Actor attempts, six CLI metadata calls, and native history of 15 starts / 16 wrappers / 13 fixture delegates / 2,250 reserved seconds.

## Authority wording qualification

`authority-binding.json` says the pinned client rechecks live grant and slot before reservation. The source ordering does not support that wording: `run_once.py` writes the exclusive reservation before it calls the controller; the controller then validates through `load_request()`, and the gated worker validates again before native launch. The root's direct prestart comparison happened before any reservation, and this grant stopped before one was written. `terminal-handoff.md` now records this distinction. It is a documentation qualification; no reservation or authority change occurred.

The local `slot-release.json` explicitly records release at `2026-10-08T15:08:27.466196+00:00` and says the grant is closed. At this review observation, canonical `coordination-state.json` still has `fullSuiteSlot` assigned to this key (file `updatedUtc` 14:59:04Z). The local release evidence is present; the Overseer-owned canonical update remains pending and is not described here as already accepted.

## Evidence limits

There is no runtime diagnostic result. New stderr observation, cause classification, thread creation, provider requests/retries, serving model, billing state, and execution timing remain unknown or not applicable. The 53,331 known historical tokens are not a total-usage figure. Zero new requested turns does not prove zero historical or background billing. The prior focused evidence is synthetic only; its final stop-event placement was statically reviewed after the last test run, not rerun. No claim follows about OS enforcement, general S1 capability, or study quality.

All six study cells remain **NOT RUN** and S1 remains **OPEN**. No repair, source change, substitute binary, or relaunch followed the blocker.

## Reviewed evidence hashes

| Evidence | SHA-256 |
|---|---|
| `authorization-grant.json` | `1d712e742f93697506b7de034d031652f33b02ceae57e4d1596ad4ace18f24cc` |
| `authority-binding.json` | `86144d8b2c64bcf8579361b8e75a6b5d4c809cfcf7a4d0f5d5d2713e4ed5ecf9` |
| `preflight-review.md` | `1ade11ebe6e45e0bd221fab692c7c08ce81410ca57b309ccf3faa3b547096881` |
| `preservation-validation.json` | `c4c08ae6c693980f3cc93be158a7c2850dab51639118045fb69995716ba5480b` |
| `terminal-summary.json` | `0064fece1af3ba00494b4b1b3f778e3a909839c41e3289c4c66bdd7f50b6647d` |
| `slot-release.json` | `3dd93f4da2a0dd59dba21b4b057a56274d84bd02b954b5274804d8c0f001c376` |
| `terminal-handoff.md` | `9f06d3491002a1351e1f8209c32820b93722a4250bf67a9a40a4d1f00ad6a66f` |
| public status report | `93c272d71f79b172ed29954f6982788847c360822b0741bf96bd35c9c6837c62` |
