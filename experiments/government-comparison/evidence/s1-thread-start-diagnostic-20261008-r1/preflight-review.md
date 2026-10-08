# Independent preflight source and privacy review

Review scope: packet `s1-thread-start-diagnostic-20261008-r1`, limited to static review of the current packet source, contracts, handoff, input binding, and offline evidence, plus comparison of the copied grant with the current canonical Scientist grant and latest coordination entry. This review does not approve or perform prestart request/freeze binding, reservation, or execution.

## Decision

No material privacy, scope, or stop-boundary blocker was found in the reviewed source bytes. The source implements the stated diagnostic-only boundary: at most one `thread/start`, no `turn/start` or tool path, and a terminal response after either the first observed stderr byte or a validated thread response. The current client SHA is `ae2101af0b052bc53002bbcad30cc9259b2f0eb672099f5ffab85c8446532b06`.

There is one exact-binding caveat for the later prestart review: `authorization-grant.json` records canonical coordination SHA-256 `0d1001b7284a1157addd36666f9850192ae992460c685abbff14eacfa448a4c5`, while the currently read canonical file hashes to `3f5efd1d15ec8bdf1db5b89a4bb927a7dbcf795e2baa14066b3d1a55059ba14d`. The `threads[name=Scientist].evidence.threadStartDiagnosticGrant` object itself matches the copied `grant` object in the authorization packet, and the latest `coordination.md` entry describes this same one-shot diagnostic. The source-coordinate digest mismatch must be explicitly reconciled in the exact prestart binding record; this review does not infer that the current source file is the issuance-time snapshot.

## Source findings

- `stderr_collector.py` enforces fixed ceilings of 16,384 captured bytes and eight complete classified lines. It retains only line ordinal, fixed code, severity, and an allowlisted component; unrecognized content stays unclassified/unknown. The bounded raw fragment is discarded during reduction or finish. No original stderr text, paths, URLs, IDs, credentials, or fingerprints are returned by the collector.
- In `client.py`, the stderr pump sets the stop event immediately after a nonempty read, before acquiring the shared admission lock. `send()` checks that same event while holding the lock. A write admitted just before the event may still be in flight, as the handoff accurately states; the implementation does not claim a stronger cancellation guarantee.
- RPC admission is ordered against the fixed method list, and `thread/start` is the last permitted write. The request loop validates the response through the pinned protocol and then `ThreadGate`; there is no turn or tool RPC in the client method set. Further RPC admission is rejected after the validated response.
- `ThreadGate` requires the reported model/provider, cwd, approval policy, effective read-only sandbox, ephemeral thread identity, no parent/fork origin, and an empty turns list. It accepts only the three declared lifecycle notifications and correlates them to the response thread. These are reported protocol observations, not proof of operating-system enforcement or tool capability.
- `run_once.py` is an explicit entrypoint guarded by `__main__`; importing it does not reserve or launch anything. The reviewed handoff and offline log state that no diagnostic process, test process, or Codex process was started during preparation. I did not invoke the entrypoint or run tests.

## Evidence limits

The offline record reports 14 initial focused cases with 11 passes and three fixture setup errors, followed by the corrected three ThreadGate cases and two cleanup-helper cases. That is 15 distinct final cases across targeted runs. The final client differs from the last-tested client only by moving the stop-event signal immediately after the nonempty stderr read; this placement was statically reviewed here, not dynamically re-executed. This is synthetic offline evidence only. It does not establish an actual stderr cause, a created thread, a turn, model/tool behavior, billing, or OS isolation.

The packet currently has no final request/freeze/reservation artifacts to bind in this source review. Clean source commit, exact source pins, current canonical grant/file binding, fresh request and freeze, historical-file preservation, and the single external reservation remain for the separate prestart-bindings review. Unknown stderr remains unknown, and any live result must retain that limit.

## Reviewed SHA-256 values

| File | SHA-256 |
|---|---|
| `authorization-grant.json` | `1d712e742f93697506b7de034d031652f33b02ceae57e4d1596ad4ace18f24cc` |
| `client.py` | `ae2101af0b052bc53002bbcad30cc9259b2f0eb672099f5ffab85c8446532b06` |
| `run_once.py` | `32bdfcc84fc25dba6d1a4fa289745674780134ee580fab9ae12b239f788fe3f3` |
| `thread_gate.py` | `b48ce8b67e3a18ecab80a49d1eb3633f90443cc9524972e00a3ea7bdf27fa6cd` |
| `stderr_collector.py` | `a9a1213146acbf31791e966992507b976ee6d3753bf50b27ecf14e7c42261df4` |
| `diagnostic-contract.json` | `0fb70bafd10b0d3de2883d1f4fcbf7cfd292a61e88a6ce524567a386e664a70c` |
| `profile.json` | `b2765de4b63224c929d55c2467d026a96a3b7d52667845c8eee916f0013ca459` |
| `schema-contract.json` | `b642953d3c663ad02df91c258b051bc5415f376c7b7e0612dcaf84a9c98c6332` |
| `current-handoff.md` | `05d55912a37d7ca88a3c5f8b86038eba37d18cec0f53aef5f09860e9b2691ca2` |
| `input-binding.json` | `5b4d2032eb6ef8c649bd2429b4e0052a65866bfe095eb3e75e5ae36b83d854bb` |
| `offline-validation.json` | `0627a7234ead573316c4e20bed2bc83f46513fd77c68f53a2161fabaac5094da` |
| `focused-tests.log` | `bcd8518209995016deec536502ca72a44df1c396fd8e38334dea71b6db9b89d6` |
| `test_client.py` | `f3f00803145ae1951157518bdacae728bda87d3be5e23f158b1ba671bc3a18bc` |
| Current canonical `coordination-state.json` | `3f5efd1d15ec8bdf1db5b89a4bb927a7dbcf795e2baa14066b3d1a55059ba14d` |
| Current canonical `coordination.md` | `4aff26155b5c052bd884080d64b33246e72170ef3a9d0b7ac2c7b42a5bd28ec3` |
