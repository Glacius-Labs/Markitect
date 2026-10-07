# Real-agent operating-model proof protocol v9

Protocol ID: `operating-model-proof/v9`. This source-only tooling amendment strengthens evidence provenance; it changes no Core, Module or provider protocol. All v1–v8 events, captures, invalid setups and failed trials retain their original bytes and interpretation. No provider invocation is authorized by this amendment.

The driver requires Python's first script argument to be the absolute repository Codex adapter path. That same path and its frozen digest must occur in the role's explicit runtime-file binding. Merely mentioning the adapter later in the argument list is insufficient. Required native-executable, version and model flags have one bounded value each. These are byte/path consistency checks, not authentication of a provider or reviewer.

Sanitized array fields carry an availability state: `missing`, `null` or `present`. An explicitly empty array is `present`. Consumers must inspect availability before interpreting an empty summary as observed absence. Counts and digests for unavailable arrays remain unavailable. Full native CLI captures stay external and are bound by their original digest; sanitized summaries do not replace them.

Apply summaries retain each returned ProjectionRecord's ID, Projection ID and bounded artifact path/mode/digest facts, with artifact-array availability. The controller's returned `evidenceRevision` remains unchanged. No artifact bytes, provider transcript or private local path is added to these public summaries.

Offline tests establish the guard and serialization contract only. Runtime-file substitution and unavailable-data controls are separate from claims about real Executor or Verifier quality. New real trials require a fresh source/binary/runtime binding and the normal reviewed write/provider boundaries. Earlier C5/C8 captures with independently confirmed aligned adapter paths are not invalidated by this guard correction; their existing evidence limitations remain.
