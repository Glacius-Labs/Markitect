# Corrected-source canary: terminal Main parser rejection

Grant `v4-canary-fixed-source-byte-correction-20261009` is **NOT READY / NOT EXECUTABLE**. The actual live transport chain ran, but Main rejected the genuine Actor response. The next concrete failure terminates this grant; there was no response repair, retry, fallback, additional test, or reviewer activation.

All 18 tracked files in the new isolated public canary repo were materialized from raw committed Git bytes. Before Main, every working `git hash-object --no-filters` matched its committed blob; local `core.autocrlf=false`. The original failed repo, archived working bytes and `6ab2669` package remain unchanged. Both transport helpers retain their exact prior raw byte hashes. No product freshness check or source was modified.

The single new Main process generated a genuine normalized Executor Invocation. One fresh native Actor inherited the parent profile (`gpt-6.1-sol` / `high`, checked from the latest own turn metadata before activation), read the actual request, authored its own candidate response, and completed. Scientist then delivered the exact response file unchanged. Main's actual Executor receipt binds the same run ID, input digest and runtime config digest; its stdout digest equals the genuine raw response hash. Thus the missing live stdin/stdout/native-tool transport was exercised end to end, including actual product parsing and receipt production.

Main exited 2 after about 77.916 seconds with `evidence reference was not supplied in the request`. Its receipt outcome is `incomplete`, retry count 0. The request has zero supplied artifacts; the Actor's proposed response includes three evidence references. The original response is retained without editing. This is a real response-validation failure and does not constitute an executable positive canary, semantic acceptance or study/product-quality evidence.

Actual counts under this fresh grant: one correction batch, zero offline test processes, one Main product process, one real Executor bridge invocation, one native Actor activation and terminal completion, one exact raw response forwarded, one actual incomplete Executor receipt. The optional final review was not reserved or started because it required successful live proof. All unused correction-grant quota is closed. Previous consumed setup effort remains one product process, two native reservations and one native reviewer activation; no old quota was reused.

Key bindings:

- Source revision: `132f8fed0871c6640af6eb6d709f9dc069a4aec5`.
- Main source: `a97cbd5ef3e0b22b9e6397501047a4e01dc90204`; binary SHA256 `0621e6ce827278d4410136cdc608286150d5d16caa566cc8d823891acc0712b8`.
- Run ID: `460e52108f199c183863f40bf2fe0768`; native Actor `/root/v4_corrected_public_canary_executor`.
- Invocation SHA256: `0607e52e6dd8728381b5487593f5904db3ad246bfa46287c4f393e13e1efbeb1`.
- Genuine response SHA256: `7a2f94ec188855435e2bbc39f353f636a5937d41e731cf191584a60070a89b59`.
- Main config digest: `sha256:df59df10bd731c303a7d49bd736addcc25ff23d1b3caf571f518a1e2aaf75aaa`.

Full raw process/request/response/terminal/delivery/forwarding evidence is in `evidence/canary/`; the actual product receipt is also in `evidence/main-executor-receipt.json`. `evidence/pre-canary-pin.json` fixes literal command, args, Python, runtime files and process spec. `evidence/source-byte-admission.json` records every raw fixed-source binding. `common-terminal-bindings.json` retains the 24 public and five private evaluation hashes and identical cell resources, with no private vectors or foreign results in this handoff. Serving model, tokens, cost and measured human time remain null. Shared-host scope remains cooperative, with no OS isolation or provider cancellation claim.

Both reserved Greenfield cells remain **NOT RUN** and Root start receipt is null. The previous pair deadline remains `2026-10-09T05:32:04Z`; no new full-trial reservation or extension was created. Any further attempt needs a fresh specific prospective Overseer instruction, not an automatic retry. This package is backed up only to PRIVATE `research-backup`, never public origin. One terminal blocker callback is due after clean commit/push and exact remote-SHA verification.
