# Independent plausibility review

Reviewed public proposal: `experiments/government-comparison/public/s1-runner-rebinding-offline-20261008.md`, SHA-256 `3e66f37e9a18cfc33b7b99b68c7ffe4a18059deb98029d5c9f58eb0fa78640a5`.

## Assessment

The proposal keeps the finding and next step within a reasonable boundary. It identifies a candidate by one exact path and observed SHA-256, distinguishes file metadata from executable version identity, and does not call the candidate the prior pinned 0.160.1 binary. My permitted read-only check of only that exact candidate reproduced SHA-256 `3553cd6e7df5a093d8cb8301cd8088a57e0971aba71ddbe0e67f7f44a15cdf68`, length `333357008`, and modification time `2026-10-07T20:15:47.4215883Z`; PE `FileVersion`, `ProductVersion`, description, product name, and original filename were empty. This does not identify a CLI version or establish command behavior.

The source/compatibility caveat is stated correctly. The available 0.160.1 schema bundle remains evidence only for that old build. A path change cannot preserve the old client pins by itself. Version-specific schema generation is a plausible next way to inspect candidate contracts, but generated schemas alone cannot prove the existing client's parser, enum tables, policy assumptions, or RPC behavior are compatible. The proposal explicitly leaves those questions open and limits any conclusion to a byte/structure comparison against named existing client/profile/parser inputs.

The official [App Server documentation](https://learn.chatgpt.com/docs/app-server) documents `app-server generate-json-schema` and says generated artifacts are specific to the Codex version used. The official [Developer Commands documentation](https://learn.chatgpt.com/docs/developer-commands) says schema generation should use `--experimental` to include gated fields and methods. These support the proposed inspection method and its experimental-field rationale; they do not show that this candidate recognizes either exact command or flag, nor that its output matches the old protocol.

## Boundaries and remaining uncertainty

The proposed two sequential argv calls are a narrow metadata/schema inspection, with explicit time, byte, file-count, hash-drift, stderr, exit-code, and no-retry stops. The output directory is described as not yet created and as requiring a later binding. The steps are proposed only: this review did not run either command, create that output directory, inspect any other binary, or run tests. No current authorization for those future commands follows from this plausibility review.

Before any separately authorized run, the exact candidate path/hash, exact argv, external output path and initial-state rule, bounded stdout/stderr and schema inventory, cleanup, and terminal stop behavior should remain frozen together. A successful schema generation would establish the reported version and schema bytes for this candidate at that time; it would not prove app-server runtime compatibility, actual RPC acceptance, thread/tool capability, OS isolation, or study readiness. If the command or flag is unsupported, or any proposed limit/identity check fails, the stated stop-without-fallback rule is appropriate.

No material flaw was found in the proposal's stated scope. Candidate version and schema compatibility remain unverified, and all subsequent CLI execution remains outside this review.
