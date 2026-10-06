# C12 goal-model proposal trial 01

Run label: `c12-goal-propose-01`
Protocol binding: `operating-model-proof/v5`, commit `658866480bafa37c8c2e94b5d58fc2074655f416`.
Technical selection: exact `unknown-ontology-mission@1.0.0#sha256:7cbb9964571e912e416ca1e56ad9e66522d1f49a36d230216a160fb4b3f374be` recommendation from `c12-goal-recommend-01`. This is an experiment selection only, not owner acceptance.

## Frozen proposal inputs

- Goal input SHA-256: `810859a4e97616dd156e998ed5f76aa7bf63fd8c6f8f8dc883ab4ecfb088e9e9`.
- Exact prior recommendation result bytes SHA-256: `7055ae20a4c35580a489f1a26768b9fb2fae8a34751cc201deb0a2100534d20a`.
- Recommendation digest: `sha256:0e240974779360e037a0d7a59f0a3d803c4ff7eadb9acbbdf77b824db0cb3f22`.
- Exact selection JSON SHA-256: `c24dcb48628bd1d54cbe2597cfa25851b4c51a6d89cf10b9502d6e33e804b44d`.
- Stage-2 runtime JSON SHA-256: `01204cf3b20e070038e266d39093ca01b989374c5535cbf44baf10be99995a26`; runner wrapper SHA-256 `7806fabc774e2127ea29977009745c49d8043ffbcba63390e3d790b82d164d57`; config digest `sha256:00954cef0ea3d2c01cc4fa6d665ee9c136abe5680acf27f26993cd82909bd868`.
- Product source/build binding: source `216e101baa2021fb6bf091206b323407aa471b26`, binary SHA-256 `bc48ac8e1792d94e7d64717b34a9c734e2b49d4da757a811add1d46192f521fc`.

## Observed result

One fresh `goal-propose` invocation completed. CLI exit code was 1 after Core compilation; elapsed CLI wall time was 28,642 ms and receipt wall time was 27,965 ms. No retry occurred. The Host retained `accepted=false` and `adopted=false` and reported `status=invalid`. Classification: **FAIL/PARTIAL for proposal compilation**; the live inference proposed a candidate, but Core rejected its reference values.

Receipt run ID: `4c23e7cc27ef6a0096dee8d946308db0`. Actual context digest: `sha256:64bc8d3a2244bddd3f48a34c1cbf49a87a66558e98e8526c5d43e16b275cfe7d`; input digest: `sha256:a3b1d6dfcd3d2b488d608254cd3dd7eac9d85c99e026ee14cdd23e8804b39d9b`. Candidate digest: `sha256:cb7fba88d49c3bd3fc7049fe1ea62c294cd6b1595822a45e9d5a54a0ca598f40`. The exact raw candidate remains external.

Core returned three `reference.value` diagnostics: each nested reference used a `metadata` object that Core rejected as an unknown field. The public Core contract requires flat `namespace` and `name`; `apiVersion` and `kind` can be omitted or supplied and checked against the schema target. The candidate therefore did not compile. This was the actual model output and diagnostics; no correction, second call, adoption, or reconciliation is represented.

The candidate uncertainty explicitly described all names, namespace, unit, requested magnitude, axis range, and capability maximum as placeholders proposed for review only. Placeholder values included unit `requested-unit`, axis range 0–100, and capability maximum 100. These are generated assumptions, not owner facts. A later technical reviewer must decide whether they are acceptable example placeholders or whether missing values require escalation. The selected schema has no decision/result Kind; the output only proposed typed facts from which later behavior could compare axis identity, unit, range, and capability maximum.

Provider-reported usage: 17,291 input tokens, including 13,696 cached; 1,312 output tokens; no tool-call count was returned. Stdout, empty stderr, exact recommendation and selection inputs, and the private runner log are retained externally under the run label. This trial establishes neither valid candidate compilation nor owner acceptance, projection, adoption, reconciliation, or verification.
