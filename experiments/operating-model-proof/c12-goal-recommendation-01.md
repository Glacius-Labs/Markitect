# C12 goal-led Module recommendation trial 01

Run ID: `c12-goal-recommend-01`  
Protocol binding: `operating-model-proof/v5`, commit `658866480bafa37c8c2e94b5d58fc2074655f416`, protocol SHA-256 `9b0c42914ddf5d2337ac6a20de0a666242d659f9c7cc92851c2f5c472d0e9635`  
Bounded claim: one fresh, read-only `goal-recommend` call over an experiment-authored public-fixture goal and exactly three supplied public Module packages. The coordinator approved this stage-1 freeze. It is technical feasibility only, not a full C12 result.

## Frozen inputs

- Product source from build receipt: `216e101baa2021fb6bf091206b323407aa471b26`.
- CLI binary SHA-256: `bc48ac8e1792d94e7d64717b34a9c734e2b49d4da757a811add1d46192f521fc`; build receipt reported `go build ... ./cmd/markitect`, exit 0.
- Runner: Codex CLI `0.130.0`, `gpt-5.5`, high reasoning, Python 3.13; wrapper SHA-256 `7806fabc774e2127ea29977009745c49d8043ffbcba63390e3d790b82d164d57`. Runtime files total 235,102,670 bytes.
- Goal input JSON SHA-256: `810859a4e97616dd156e998ed5f76aa7bf63fd8c6f8f8dc883ab4ecfb088e9e9`.
- Runtime-envelope JSON SHA-256: `180b61f2d01be237e2cb0780dccb29257ae0d5d873fb99bed3a084f8b7e6a44d`.
- The supplied package pins were Foundation `sha256:f6e1fff3e8ba4ca0336ccdabd74196595f8cd37778d831cffa655eb35fcb2b7b`, Mission `sha256:7cbb9964571e912e416ca1e56ad9e66522d1f49a36d230216a160fb4b3f374be`, and .NET `sha256:14c81a45a1ff14da5f3b736cbf2ba8b63382b0e5cd89d1ff30e44ee9e2ea1fad`. The embedded exact bytes were checked against these public fixture pins.
- Goal digest `sha256:e1246709c39b8868cf159f233519901c538da036e963b1851708c9cc01831252`; catalog digest `sha256:0b7cd9e45bc98a4ec09a8f15403b057c50e0035259ef303f4311fb3e9edce049`.
- The pre-call offline request/context digest calculation was incorrect: it recorded context `sha256:a995bdc8901c7ade3bc69d6e3db82e431889aad0da15a6a2e43481285f995fdd` and input `sha256:dbe0095382b9aec555bc08a77fe6ce4592eac94c79804897b6d0d97c8b332913`. These values are retained as the original freeze record and are not replaced with post-run values.

## Observed result

The frozen binary made one native fresh `RoleInfer` call. CLI exit was 0; elapsed wall time was 20,935 ms, receipt wall time 20,297 ms; retry count was 0. Host outcome was `proposed`; `accepted=false`, `adopted=false`. The response contained one unique recommendation, within the exact supplied package set: `unknown-ontology-mission@1.0.0#sha256:7cbb9964571e912e416ca1e56ad9e66522d1f49a36d230216a160fb4b3f374be`. Its basis and uncertainty were nonempty. No IDs were unknown or repeated. The recommendation digest was `sha256:0e240974779360e037a0d7a59f0a3d803c4ff7eadb9acbbdf77b824db0cb3f22`.

Actual receipt bindings: run ID `36cb8947968a47484e877a833112b799`; goal digest and catalog digest matched the frozen values; context digest `sha256:f50da2b3b33a72344b1a11840ae1e59eba937f36245442d80adc5b87438c1e28`; input digest `sha256:a343b609b844a9f05f316eefc014b68cf724b02f1094acf1a2d8710ead86f490`; config digest `sha256:00954cef0ea3d2c01cc4fa6d665ee9c136abe5680acf27f26993cd82909bd868`; runtime-files digest `sha256:de49d71a392a12d6c786c91683d5bbd6de7e447bcfe45f7367b379690f71dbc7`. The runner receipt records stdout digest `sha256:ddc700ba4b47a41074d18e001a553cb4b1d1bd9b760ec01f40fc3f97a3286427`, empty-stderr digest `sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`, and private-log digest `sha256:595f9bfabf4be4238e080ce9653f87621e73ca7d58e865078e63d84c74e7014e`.

Provider-reported usage: 19,179 input tokens, 13,696 cached tokens, 898 output tokens; tool-call count was not supplied. Raw stdout, empty stderr, runtime/input files and private runner log remain in the external run store; this record omits raw prompt, provider log and recommendation rationale text.

## Classification and next decision

Stage-1 output is structurally valid and bound by the actual receipt to the exact goal/catalog and runtime. The pre-call context/request digest mismatch is a provenance limitation, so this run is recorded as **PARTIAL** for the planned pre-frozen stage-1 experiment. The discrepancy is not repaired after the call and no retry is authorized by this freeze. This does not establish provider recommendation quality, user effort reduction, owner acceptance, schema proposal quality, Core compilation, reconciliation or verification.

The exact recommendation ID above is available for the coordinator's technical experiment selection. `goal-propose` has not been invoked. Any subsequent proposal must be a separate fresh call after the coordinator explicitly selects IDs; that selection is not human architecture acceptance.
