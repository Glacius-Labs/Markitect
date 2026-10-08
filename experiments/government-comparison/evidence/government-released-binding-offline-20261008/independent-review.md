# Independent review: released-input binding correction

**Disposition: bounded offline producer/consumer regression is sound; R5 remains NOT ADMITTED.** The correction preserves the distinction between the exact file released as an input and the grant metadata carried by the Request. This review covers only the helper, its focused regression tests, the adjacent preparation shape, and the existing validator. No preparation, validation, freeze, test command, ledger access, or process start was performed during this review.

`run-native-integration-r5.py` now centralizes construction in `fixture_grant_bindings()`. It emits three canonical released-input records, each containing only `{path, sha256}` for the original grant, successor grant, and coordinator snapshot. Separately, it emits the original and successor Request metadata with the required `sourceKey`. `prepare()` uses those returned values consistently: the successor metadata is still the `nativeFixtureR5Grant`, and the same metadata object is carried into both the Protocol and Grant as `fixtureR5SourceGrant`.

The consumer in `runtime/native_fixture_budget.py` is unchanged from baseline `8702d6ad0f8eed724b77acd7e827880d171e7703`. Its exact-key checks remain strict: grant metadata requires `{path, sha256, sourceKey}`, while `_released_binding()` accepts the matching path and digest only from a separate released-input item with exactly `{path, sha256}`. Source snapshot identity and the original-grant identity checks also remain in place. `prepare-native-r5.py` is unchanged and already contributes the R5 released file as a two-field path/digest record while preserving the richer `correction` object in `nativeFixtureR5Grant` metadata.

The new tests call the real `validate_r5_grant_binding()` consumer; they do not mock or replace it. Inputs are copied into a temporary directory after checking their sealed SHA-256 values. The test relocates the coordinator snapshot path in a copy of the envelope and re-encodes that envelope for the temporary fixture, while retaining the sealed grant and snapshot contents and patching only the associated location/digest constants. `Popen` and `sqlite3.connect` are patched to fail if used. The tests cover the production-shaped successful binding and rejection of the old three-field released item, missing or different successor releases, wrong released-file or metadata digest, wrong source keys, missing/mismatched snapshot release, and violation of the original-grant two-field release contract.

The retained final focused log records **7/7 passing tests** (`focused-tests-2.log`, SHA-256 `1c8935a39fa93c6823a862be8587ace0d03d5b8d9f6487b25f4e597c4fb5c4f9`). The earlier temporary-path assertion failure remains preserved in `focused-tests-1.log`; it does not replace the final passing log.

Reviewed source hashes:

- `run-native-integration-r5.py`: `6749067ad68e6dad30f020e8cee2bc105f63717db6fda4eb4369c882634a65fc`
- `runtime/test_government_released_binding.py`: `09d4774a809c60ef3b60d6df14dcbbfb55a630600de589fc703c7f740ff1e34c`
- unchanged `runtime/native_fixture_budget.py`: `6a2eec833affad4cc9cf1a971bc17992759a29ca235c90d1ac038467f9898971`
- unchanged `prepare-native-r5.py`: `d3d5a7dc57ec4ced26b204a3a7f578ce1ef4fa0493e10fb64fa6584355b053e4`

This corrects the Request producer's released-input representation, but proves only the isolated binding contract. R5's previous actual Request failed preflight and remains terminally not admitted; no new Request was prepared or validated, and no reservation or experiment was created. The controller deadline/terminal receipt-write behavior remains unproven by native execution. No product, S1, semantic, or human-acceptance claim follows from these offline tests.
