# Redacted stderr consumer wiring: offline only

Assignment `s1-redacted-stderr-wiring-offline-20261008`, base `3f91eeefaad0191d76af3363a9165f1fe719f78c`. This package closes the standalone collector's consumer gap with an additive copy of the sealed R2 client. It grants **zero runtime starts** and creates no session reservation, session freeze, profile or executable grant. Previous R2 client/collector and every prior evidence package remain unchanged.

## Concrete wiring

The new `client.py` retains the R2 RPC sequence, metadata/schema consumers, thread gate predicates, model/provider/policy/identity checks, finite deadlines, Windows Job controller and live grant/slot validation. An additional constant `OFFLINE_WIRING_ONLY=True` fails closed at CLI entry, pre-reservation validation and `load_request()`, before any process can start. The retained R2 key/issued date identify derivation, not reusable authority. Tests do not switch that constant or weaken/replace live authority checks.

The copied `session()` constructs `StderrConsumer(stderr_seen)`. It hash-loads the accepted collector and gives it **the same event object** that the unchanged `session.send()` passes to `admit_rpc()`. `pump()` calls `observe_read()` immediately after stderr `read1()`: it sets this event before taking the RPC admission lock, then invokes the existing consumption boundary with the actual `cleanup_started.is_set()` flag. The first nonempty read is terminal even if every text line is suppressed. A write already admitted before the signal may still be in flight; this existing race boundary is unchanged.

Once stderr has triggered, the pump waits for the existing cleanup event before reading any continuation. There are no new RPC, turn, task tool, interrupt or retry paths. The existing native wait/kill and finite pump joins precede `finish_after_cleanup()` under the same lock. The function allows a private file only when cleanup started, the actual native object reports an exit via `poll()`, every pump reports `is_alive()==False`, the cleanup deadline has not expired, and stderr really triggered. A missing/failed check returns fixed metadata such as `not-written-not-quiescent`, `not-written-cleanup-deadline` or `not-written-stop-capture-incomplete`; it does not finalize mutable collector state while a pump could still own it.

The output attempt is once-only and cached, including failures. After the lifecycle checks, the consumer creates the private child directory exclusively and invokes the unchanged accepted writer. `stderrDiagnostic` in the normal result contains only the collector's metadata and `excerptStatus`. Neither excerpt text nor its content hash appears in normal results, controller receipts, archive manifests or callback. All-sensitive input can still produce a private file containing drop metadata and an empty text list; it still represents a terminal stderr stop. A failed disk write may leave an empty/partial sanitized file: the explicit state is `private-output-failure-artifact-may-remain`, not a claim of absence. Only `written-private` permits a later targeted read. The future task must remove its owned partial artifact at closure as well, without retry; preexisting directories/files are preserved.

## Prepared destination and future authority

Proposed later grant key: `s1-redacted-stderr-diagnostic-20261008-r1`, **not authorized or allocated here**. Proposed new external evidence root:

`C:/Users/Consiliari/Documents/Scientist-Probes/s1-redacted-stderr-diagnostic-20261008-r1`

Private output is exactly:

`private-redacted-stderr/redacted-stderr.json`

The root must be newly and explicitly bound by that later grant; the consumer exclusively creates the previously absent private subdirectory, outside the repository. An existing child is preserved and stops the output attempt without overwrite or retry. No proposed directory or real excerpt is created in this offline package. Tests use only temporary directories with harmless synthetic text, removed by their fixture cleanup.

A future runtime preparation must explicitly rebind the proposed or Overseer-selected **fresh** key, issued date, canonical live grant/slot pointer and status, source commit, binary/schema/interpreter/profile/input pins and new Request/Freeze/reservation. It must supply the complete existing R2 gate/schema/enum dependencies in its fresh source binding. No R2 reservation or closed quota can be reused. These are preparation requirements, not placeholders accepted as live authority. The current copy deliberately cannot run and has no session Request/Freeze. Architect retains the heavy slot; a later finite grant and explicitly assigned free slot are required before any actual attempt.

## Preserved observation contract and limits

The accepted collector source is unchanged: first four physical complete lines, at most1,024 input bytes including delimiters; invalid, incomplete, cap-cut, known-sensitive, malformed/escaped or non-SGR control content suppressed; URL/email/absolute and known paths redacted. Private JSON is at most2,048 UTF-8 bytes including metadata, with whole-record output fallback. No raw stderr/frames or their content fingerprints are written. No collector-suite rerun, new parser, DPAPI capsule or human-reader requirement is introduced.

Only a later deliberately targeted local tool read may place the **sanitized** file in Scientist/Overseer model context. The local file is deleted after diagnosis/closure, but sanitized task/model history can remain. The excerpt must not be copied into public Git/PRs/callbacks or product-implementer handoffs. The future grant owns private destination access and removal; neither writer nor consumer proves ACL/OS isolation. Pattern-based redaction can miss unknown/obfuscated/context-sensitive secret forms, and conservative suppression/caps can lose useful detail. Filesystem I/O is not an independently preemptible timer; the existing process/Job deadline remains the enclosing runtime bound.

This package demonstrates synthetic consumer behavior only. It does not diagnose R2's lost lines or prove actual RPC transport, process-tree shutdown, runtime permission/tool capability, root cause, provider usage or billing. Categories remain reported-message descriptions with line number and uncertainty, inferred only from a future actual sanitized read. All six study cells remain NOT RUN.

Classic product main integration does not change frozen study pin `c91363b7ac4decbe87212ff0f588b5451581a152` / v0.14.1 or its inputs. This package neither reads other cells nor forwards private comparison results. The reported main PR/CI state is outside this wiring check; no new claim of current CI completion is made.

## Focused evidence

Six synthetic Consumer methods passed: stop signal before the blocked admission lock and next RPC refusal; cleanup-only fragmented line; readable private file/text-free normal receipt; sensitive whole-line suppression; no-trigger/not-started/alive-pump/running-native/expired-cleanup no-write boundaries; and bounded output with turn/tool refusal. The initial run passed6/6 in0.056s (wall0.182s). After the write-failure status qualification, only the two affected methods reran:2/2 in0.036s (wall0.261s), adding fake `read1()` chunks, once-only/cached output, injected harmless partial-write failure, and preservation of a preexisting private artifact. No other suite, process launcher, reservation or live authority was exercised or patched. Source wiring into `session()` is independently reviewed; fake lifecycle checks do not prove live transport. Exact pins/history: [derivation](derivation.json), [offline validation](offline-validation.json), [test log](focused-tests.log), [independent review](source-result-review.md).
