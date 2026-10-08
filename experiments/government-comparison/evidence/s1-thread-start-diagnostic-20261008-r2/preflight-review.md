# Independent source preflight review

## Decision

I found no material scope, authority-flow, identity, or RPC-stop defect in the reviewed source. This is a bounded static review of the current source and proposed profile only. The final source commit `S`, request/freeze records, exact live binding, and current pre-reservation byte checks are pending; this document does not certify those bindings or authorize execution.

## Reviewed source snapshot

The reviewed packet is `s1-thread-start-diagnostic-20261008-r2`. SHA-256 values at this review are:

- `client.py`: `85a3181560c97e150d52fbb12458beaa3958993ffaecbdcffa077ff14dc30bb5c`
- `run_once.py`: `55d4321c9f315b93a986ec17fc23cf60f68d4296dbc2488e05620dd01ce5f096`
- `profile.json`: `1f8eb50373349a0c46c9b5a795638aa5e0d6ab66fff029019194e9913112f188`
- `schema-contract.json`: `7880a5d32f71ddb09258dcb829ef6ec0e9384e0c06f062c99a7a585143458d48`
- `authorization-grant.json`: `ce85999beae78dde3d7f26d153e0dfde31d0f0ab876e9630f1d8e4620df34a61`
- `input-binding.json`: `03355c2e1eb0fefd666347beaca776eecb37279e562dfe6f596341749de644a86`
- Reused `thread_gate.py`, `stderr_collector.py`, and `diagnostic-contract.json` retain the R1-bound hashes `b48ce8b67e3a18ecab80a49d1eb3633f90443cc9524972e00a3ea7bdf27fa6cd`, `a9a1213146acbf31791e966992507b976ee6d3753bf50b27ecf14e7c42261df4`, and `0fb70bafd10b0d3de2883d1f4fcbf7cfd292a61e88a6ce524567a386e664a70c` respectively.

## Scope and stop behavior

The copied grant binds one exact executable and schema archive, unchanged profile inputs and 17 config pairs, the same metadata handshake and thread-start payload, one app-server tree and one diagnostic reservation, zero Actor/task reservations, and zero turns, tools, metadata calls, retries, parallel work, or study cells. The run path calls `validate_pre_reservation()` before exclusively writing the reservation. That function compares the copied grant and slot with the live coordination objects, validates profile/grant identity and limits, checks the candidate and schema hashes, and validates the two-file public cwd inventory. The controller then calls `load_request()` before starting its gated worker; request, freeze, repository/source pins, interpreter, historical ledger pins, and live authority are checked there again. Thus the exact request/freeze check is pre-worker but occurs after the one diagnostic reservation; the exact final S and binding remain for the separate binding review.

The client sends only `initialize`, `initialized`, `config/read`, `configRequirements/read`, and `thread/start` in fixed order. RPC admission checks the first-stderr event and the validated-thread-response state under the same lock as writes. The stderr reader sets its terminal event immediately after a nonempty read and before acquiring that lock; a write already admitted may still be in flight. The unchanged strict thread gate requires the requested thread identity, effective read-only sandbox, ephemeral state, no parent/fork, and an empty turn list. The client has no code path for a turn, tool, or follow-on request. New prediction notifications and every unlisted method, server request, error/warning/auth/approval event remain terminal.

The bounded reader, frame/notification caps, finite schema validation, and sanitized result fields retain the earlier diagnostic's privacy boundaries. Raw server frames and stderr are not persisted; stderr remains capped at 16 KiB and eight complete classified lines with unknown causes unclassified. The Windows Job is assigned before worker GO and bounds process lifetime. It does not prove filesystem or network isolation, absence of native internal inference, or provider usage/billing state.

## Inherited classifier provenance

`diagnostic-contract.json` is intentionally byte-identical to the R1 contract. It still contains R1's grant key/base SHA and an R1 binary hash for the static component-string provenance. The R2 execution path does not load that contract; it imports the unchanged collector and classifier logic. Treat those fields as inherited R1 classifier provenance only: they are not R2 authority bindings and do not prove that the new executable emits those component tokens. No new-binary component-compatibility claim is made.

## Pending checks and limits

The exact final `S`, request/freeze/reservation bytes, source pins, candidate/schema/input hashes, current live grant/slot equality, and output-root initial state have not yet been independently bound in this review. The external root was prepared with an absent-before-creation record and zero diagnostic calls, but its final contents and request/freeze binding must be checked against the final source before any start. This report does not establish RPC behavior, actual thread response, OS enforcement, a historical cause, model quality, provider usage, or billing. I ran no tests, processes, or Codex commands.
