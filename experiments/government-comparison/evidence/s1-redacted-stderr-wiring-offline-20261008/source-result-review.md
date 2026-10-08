# Independent source and result review

## Assessment

The additive R2 client copy wires the accepted redacted collector into the actual session stderr pump while retaining the same RPC admission event and lock. The private-file writer runs only after the copied cleanup path has closed the native process and quiesced every pump before its deadline. The result metadata does not contain excerpt text or a content hash. The offline-only guard remains enabled, and this packet has no request, freeze, or reservation. I found no material scope or lifecycle defect in this bounded offline change.

## Source and test pins

The final hashes match `offline-validation.json`:

- `client.py`: `ac665dc546e7e333fcaf0e97e011e22b606d8809df2e47d23fdddb351d3e1dc8`
- `test_consumer.py`: `16413873652d005fd719798fda57135ba01b81f054a51223a8a2188efe4b987e`
- `contract.md`: `25e0d699ca466b30c1e44223405b3c1a63ca0fe3b6a30f88721450b824854098`
- `derivation.json`: `25af0e7e88a8bffd7f9463276b83d418b7ff945731a7d7d206ce0d1707c92ca8`
- `focused-tests.log`: `51867d5d461f3890be56f490493580808e176617e41bf40c3310cd0fa4adc566`

The derivation identifies four changed existing functions (`consume_stderr`, `load_request`, `session`, `validate_pre_reservation`) plus `StderrConsumer` and the offline guard; it reports 27 unchanged original functions, including admission/controller. The accepted collector is hash-loaded at `d92f79b0d14abd2c5851bea2162813c5f93b396e4d19fc927389bc8b6557ef6d`. The focused history records six synthetic consumer methods passing, then one follow-up execution of the two affected output/admission methods after the write-failure status clarification. Tests used synthetic input and did not switch off or replace live authority checks. I did not run tests.

## Lifecycle and output boundary

`session()` passes `stderr_seen` to `StderrConsumer`; `session.send()` checks that same event under the RPC admission lock. After `read1()`, the pump sets the event before waiting for that lock, then supplies the actual cleanup flag to the collector. Thus the first nonempty read is terminal, while a write already admitted may be in flight. Continuation reads wait for cleanup. No turn, task-tool, interrupt, or retry path was added.

The output gate requires cleanup to have started, the native process to have exited, every pump to be dead, and the cleanup deadline still to be open. It is called under the shared lock after the native wait and pump joins. On a failed gate, no collector finalization or file write occurs. The result contains only fixed metadata and `excerptStatus`; successful excerpts are written once to the grant-bound external root under a fresh private child directory, and only `written-private` permits a later targeted read. The cached receipt prevents retry. A failed disk write may leave an empty or partial sanitized artifact; the contract records `private-output-failure-artifact-may-remain`, forbids false absence claims, and requires owned cleanup at later closure. Existing private directories/files are preserved without overwrite.

The normal receipt never receives excerpt text or a content hash. The future task-history boundary is explicit: a targeted sanitized read may remain in task/model history after file deletion. The writer does not establish an ACL or OS-isolation guarantee. Known-pattern redaction can miss unknown or obfuscated secrets; no real stderr or actual diagnosis was produced.

## Execution boundary

`OFFLINE_WIRING_ONLY` is true and fails closed at CLI entry, pre-reservation validation, and `load_request()`. No request, freeze, reservation, or runtime allocation is present in this packet. The six consumer tests and two targeted reruns demonstrate synthetic consumer behavior only; they do not validate a live transport, actual process-tree shutdown, model-read handling, permission/tool capability, provider usage, or billing. The existing R2 client and collector remain unchanged, and any future use still requires a fresh explicit grant, source/request/freeze bindings, and an assigned free slot.
