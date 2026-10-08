# Independent prestart review

## Decision

No material source, authority, deadline, or privacy blocker was found in the final bound recorder. The proposed operation remains exactly two sequential candidate CLI metadata calls at most. The first is a bounded version query; the schema-generation call is reachable only after a valid bounded version line, exit 0, empty stderr, and stable binary hash. Any first-call stop closes the remaining allocation. This review did not invoke the recorder, candidate, tests, or any Codex command.

The result is a prestart source-and-binding review only. It does not establish that this candidate accepts the proposed commands or flags, identify its version, establish schema compatibility, or approve later RPC/runtime work.

## Exact binding checked

- Grant: `s1-runner-rebinding-metadata-20261008-r1`, issued `2026-10-08T15:29:49Z`, based on `2e1d41bd120eda4b83336c3b5ad9fdaf4dda72f2`.
- Source `S` is `e67bcbe55dc266b02ce669106e44459580d19c9f`; it equals current HEAD. All 13 `binding.json` source entries matched their current SHA-256 values; no source file was changed. The only current worktree addition at review time was the new `binding.json`, pending the documentation commit.
- `binding.json` SHA-256: `0ed6493b1abb9c6307dfad747c89ff853cac796755f4231c0bf29ec119fcb29a`. Its `sourceCommit`, 13 source paths/hashes, interpreter path/hash, and allocation limits were read directly.
- The copied grant and slot in `authorization.json` compare equal to the current canonical `Scientist.evidence.runnerRebindingMetadataGrant` and `fullSuiteSlot` objects. The live slot remains assigned to this exact key. The canonical file hash equals the preparation-time hash recorded in `authorization.json` (`9734e60e9cf847b93f96544ab0c90a7821a0440a234e9fd84ddee76c8208d283`).
- The candidate was checked only at its authorized path. Its current SHA-256 matches `3553cd6e7df5a093d8cb8301cd8088a57e0971aba71ddbe0e67f7f44a15cdf68`. The version is still unknown; PE version fields were empty in the prior narrow inspection.
- The external root contains only `preparation.json`, whose `codexInvocations` value is zero. No invocation marker, schema output, or runtime receipt is present. The preparation file binds the existing public two-file cwd hashes and the candidate hash. The output subdirectory does not yet exist.

## Source and boundary review

The recorder SHA-256 is `7f8cefcd3f894c9b18695db4756b9423d02f9032db97513b1a457862e128e9d9`. It checks source pins and live grant/slot before each call, checks the exact binary hash before and after each call, and checks the public cwd inventory before and after. Its exact argv source is the bound grant:

1. `C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/9691020b546a15b2/codex.exe --version` — 5 seconds.
2. `C:/Users/Consiliari/AppData/Local/OpenAI/Codex/bin/9691020b546a15b2/codex.exe app-server generate-json-schema --experimental --out C:/Users/Consiliari/Documents/Scientist-Probes/s1-runner-rebinding-metadata-next/schemas` — only after a positive first result; 20 seconds.

The controller stops its worker at 40 seconds and bounds owned cleanup/readback within 45 seconds, with cleanup capped at 5 seconds. It records at most two invocation attempts and has no retry or help fallback. It accepts only a narrowly formatted version line, captures at most 256 version-output bytes transiently, discards schema stdout and raw stderr, and treats any observed stderr as terminal. It checks the schema subtree for JSON-only entries and observes 32 MiB / 2,048-file limits, recording overshoot. Those polling checks are explicitly not a hard disk quota. External-root contents are checked at startup and before each invocation.

The process Job is assigned before worker GO and closed to terminate surviving descendants. That controls process lifetime, not filesystem isolation, config/credential access, networking, or native internal work. The recorder does not inspect or persist credentials or raw logs. Its handoff states that no global side effect is OS-enforced and leaves native inference unknown; I agree with that qualification. Nothing here proves that the CLI commands are side-effect-free outside the bound schema output directory.

The official [App Server documentation](https://learn.chatgpt.com/docs/app-server) describes version-specific generated schemas. The official [Developer Commands documentation](https://learn.chatgpt.com/docs/developer-commands) says `--experimental` includes gated methods and fields in generated schemas. These facts support the proposed metadata inspection, but do not prove that this candidate implements the exact command/flag or matches the existing client contract. The final follow-up must compare actual generated schema bytes and referenced types/enums against the named old parser/profile/client inputs and report equal, changed, or unresolved. It must not infer runtime or tool compatibility.

## Execution state and limits

This is a review of preparation only. The exact proposed CLI calls remain unexecuted and no further test was run. The grant authorizes no app-server session/RPC, Actor, thread, turn, tool, model, product, wrapper, delegate, or study-cell operation. Candidate support for `--version`, schema generation, and `--experimental`, schema compatibility, and any native activity remain unobserved. The proposed calls require their own complete terminal accounting and an independent postreview; this document does not authorize them.

## Reviewed SHA-256 values

| Input | SHA-256 |
|---|---|
| Public offline proposal | `3e66f37e9a18cfc33b7b99b68c7ffe4a18059deb98029d5c9f58eb0fa78640a5` |
| `authorization.json` | `4ade0a4a9818ceaeda581a74840b333228aace12fb891fc6358f6d925cb72677` |
| `binding.json` | `0ed6493b1abb9c6307dfad747c89ff853cac796755f4231c0bf29ec119fcb29a` |
| `preparation.json` | `96cd945f1c9e52fad9ead64bb6fc1927261c4fb8b4699d8196dee6d36403e3ad` |
| `recorder.py` | `7f8cefcd3f894c9b18695db4756b9423d02f9032db97513b1a457862e128e9d9` |
| `current-handoff.md` | `ce6a765ab7780e6155ffc8816b2d5fb715089288cbcb9c5c87730a337ce3510b` |
| Candidate executable at checked time | `3553cd6e7df5a093d8cb8301cd8088a57e0971aba71ddbe0e67f7f44a15cdf68` |
