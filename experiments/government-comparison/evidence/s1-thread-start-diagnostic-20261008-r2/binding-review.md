# Independent exact binding review

## Decision

The final R2 source, request, freeze, current grant/slot, candidate, archive, input, and initial external-output state agree with the reviewed allocation. I found no material binding mismatch. This read-only review occurred before reservation and before any native start; it does not authorize or perform the diagnostic.

## Exact source and request/freeze binding

- Source `S` is `c6d84de416986837adfa57967c8b9ef29a1eb90a`, and it equals HEAD at review. The request names this same source commit. Its SHA-256 is `5780b4a749f766962b09ef4ec89f1ca1b38db20101017440941ed658f0469dbf`; the freeze SHA-256 is `9a189e4baec5024d2941d27020180520c10f411ffb2ae09127f4fa20312a46ed`.
- The request has 22 source pins; I checked every path against its current file bytes and found no mismatch. The freeze has 32 file pins; every pinned local or external input was present and matched. The two newly prepared request/freeze files were the only untracked paths at review; no source edits were present.
- The current live `Scientist.evidence.threadStartDiagnosticR2Grant` and `fullSuiteSlot` compare equal to the copied grant and slot in `authorization-grant.json`. The recorded coordination-file SHA also matches the current file SHA. The grant is assigned to this exact key, one prospective thread start and one diagnostic reservation, with zero Actor/task reservations, turns, tools, CLI metadata calls, retries, and study cells.
- The request binds `profile.json` SHA-256 `1f8eb50373349a0c46c9b5a795638aa5e0d6ab66fff029019194e9913112f188` and `authorization-grant.json` SHA-256 `ce85999beae78dde3d7f26d153e0dfde31d0f0ab876e9630f1d8e4620df34a61`. The actual `client.py` SHA-256 is `85a3181560c97e150d52fbb12458bea3958993ffaecbdcffa077ff14dc30bb5c`, matching current bytes and the source/request/freeze pins. The earlier preflight report contains a typographical extra `a` in this hash; this binding review records the correct value without modifying the immutable source review.

## Candidate, schema, inputs, and output state

- The exact authorized candidate at `C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/9691020b546a15b2/codex.exe` hashes to `3553cd6e7df5a093d8cb8301cd8088a57e0971aba71ddbe0e67f7f44a15cdf68`.
- The schema archive hashes to `3157e8a55329cf4e5346676c3ef0c308402d2420f8f916d6a5b7e0e9c84356ad`, matching the grant. The 12 member hashes in `schema-contract.json` were checked directly against the archive; none were missing or mismatched. These are the complete repinned members used by the existing protocol contract.
- Both public cwd inputs (`probe-input.txt` and `README.md`) match the profile hashes. The freeze also pins the selected interpreter and five historical ledgers; all freeze entries matched as noted above. The external evidence directory exists and is empty: no reservation, invocation marker, or output artifact is present.
- `run_once.py` performs the exact live grant/slot and profile/candidate/schema/cwd checks in `validate_pre_reservation()` before writing the exclusive reservation. It rechecks the complete request/freeze/source/interpreter/ledger binding in `load_request()` before starting the gated worker. Thus the full request/freeze byte comparison is pre-worker, after the one diagnostic reservation; the separate source preflight describes that ordering.

## Scope and limits

This review checked hashes and JSON authority objects only; it did not run tests, processes, Codex, or any native command. No reservation or start had occurred at review time. The binding establishes intended inputs and allowed scope; it does not establish runtime RPC behavior, actual thread response, OS-level isolation, provider inference/usage, or billing. The carried R1 diagnostic contract remains provenance for the unchanged classifier only and does not prove the new binary's emitted component strings.
