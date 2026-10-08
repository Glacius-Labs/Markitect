# R6 independent source preflight review

**Disposition: source review finds the bounded R5/R6 implementation substantially closed, with one admission-boundary issue to resolve before the authorized actual-input validation. No preparation, validation, freeze, test, ledger, native, wrapper, delegate, or provider operation was run by this reviewer.**

The new R6 identity matches the frozen grant envelope and coordinator snapshot: key `government-released-binding-native-20261008-r6`, dispatch `government-native-released-binding-r6`, task `positive-overflow-release-r6`, marker `nativeFixtureR6Grant`, metadata `fixtureR6SourceGrant`, source pointer `threads[name=Scientist].evidence.governmentReleasedBindingNativeGrant`, and the exact assigned Scientist slot. The profile fixes distinct R6 paths and digests. The shared selector admits exactly one profile marker matching the dispatch and Government arm; R5, R3, R4, mixed markers, and unrelated dispatches do not fall back into R6.

The preparation path reuses the selected profile for the output root, trial/dispatch/task IDs, grant marker, and runtime identity. It adds the profile module to the executor, verifier, and configured ressort runtime files before serializing the runtime; the selected task ID is passed to `build_backlog()`. Released grant/snapshot inputs remain canonical two-field path/digest records, while the grant marker and Protocol/Grant metadata retain their separate source key. The actual validator checks the original R1 grant, exact successor envelope and source snapshot, pointer/thread/grant identity, task/base/product/delegate/Python pins, quota and history values, and active live grant/slot/time. Budget admission preserves the immutable twelve-row history and prior corrections, uses only the selected profile's Queue then conditional Resume labels, and applies the exact cumulative ceilings without refill or cross-profile fallback.

The R6 one-shot validation action creates its durable claim before performing the full actual Request/Authority, grant, product, role authorization, slot, and read-only ledger checks. Later freeze checks the successful receipt and captured hashes rather than calling the R6 validation action again. Freeze binds the receipt, source/runtime pins, Request/Authority, grant/snapshot, external inputs, and current source state. Queue and Resume gates require full positive native and outer results before Resume; completion timing is measured after receipt/result readback and must remain within 38 seconds. Failure paths retain consumed reservations and do not provide repair/retry behavior.

**Admission issue:** after the one-shot validation receipt, runtime dispatch still calls `Authority.validate(raw)` in `dispatch_government()`'s `native_fixture_start_gate()` before the native queue, and the Resume driver calls `auth.validate(raw)` immediately before its launch gate. These are repeated full Request/released-input validations after the explicit actual-input-validation action. The grant says one actual input validation and permits only hash/live-slot rechecks afterward; the current contract does not explicitly exempt these dispatch-time `Authority.validate()` calls. I therefore cannot certify the single-validation limit as written. Before actual validation or freeze, the Overseer must decide whether those ordinary per-dispatch checks are outside the grant's count, or require a narrowly bound reuse path that relies on the one-shot validation receipt while retaining fresh hash and live-slot checks. No such interpretation or code change is assumed here.

The retained offline logs report 20/20 R6 producer/checkpoint tests, 6/6 R6 budget tests, 6/6 shared R5 budget regressions, and 10/10 R6 chain tests. This reviewer read the logs and did not rerun them. These results do not establish a fresh fixture, actual Request validation, native execution, S1, product quality, or human acceptance. R5 remains terminally NOT ADMITTED; R4 remains incomplete. Controller deadline enforcement and terminal receipt writes remain source-level claims until the specifically authorized run.

Reviewed source SHA-256 values:

- `prepare-native-r5.py`: `16c1285e704312d36c61515d54bd35c54780d22d4c08b093252717349091fd95`
- `run-native-integration-r5.py`: `8883a8cab98d0e9419921c792db3e1c74b610f3bcb1bcfd3cc6f64656c7e9c42`
- `runtime/government_native_profile.py`: `08d96a67ddd6e5a0aaf125be15084e1cce9d6af2218ed166181fedd14e2afe3e`
- `runtime/native_fixture_budget.py`: `35a33181c52c851a6e04495badb845cd532e46acada5e462d5a56433bad71c44`
- `runtime/native_controller.py`: `3c96413ab17acb13031a8a2d04f59e5cebcdeb7ea3fc36c4708fa56cea87d412`
- `runtime/government_roles.py`: `4910082ad9889eeed2f71182724ea22797c6d018fd127c7cf6837a596b0cd9aa`
- `runtime/dispatch.py`: `9375d91264aa620a570eb80052e6b2b8f5cd4a7b0dd65735d2fd9743eb1a098e`
- `runtime/test_government_released_binding.py`: `656a240137a5599e07828b89d1576b1b4f9661cef885131dc2da5765e3465b2b`
- `runtime/test_native_r6_budget.py`: `b91d1c165795289eb997c72271b7a4352e364c4abc4985f304591a588bc67e8e`
- `runtime/test_native_r6_chain.py`: `72d2e46d7f73982afb81ffaa4eb87a3ed2486885cbab7cffca6ffc73bd16ea59`
- `runtime/test_native_r6_checkpoint.py`: `fe82136db65d53da6e5511d768c3bd98dd26d973fe01f6784e170ddd8da5112a`

The R6 grant and coordinator snapshot hashes are `257ac1c4e7ebd2acb7024e3286b5de1871064def856406a1726d788cf2701721` and `15a996adf7b532f01497fa3e6f47673a0430a29fc9143270c59c0fb48c2850cc`. The evidence ledger snapshot hash remains `dd617d58a9021fce0b11482b740d78ccb5ddd143ff0c6fa4fbe08983af7f6705`.
