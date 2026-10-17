# Codex image metadata preparation

This package implements Apply's static `AgentProfilePreparer`. It does not implement `run.Sessions`. Bootstrap injects `NewEngineImageReader` into `NewMetadataPreparer`; the factory receives its executable only from the selected, source-bound Host section. It has no PATH, environment, desktop binary or historical offline fallback.

The existing Host envelope remains `markitect.host/v1` with a `codexAppServer` section. The closed section has these required fields:

```yaml
profileVersion: 2
engineExecutable: /usr/bin/docker
imageID: sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
agentdVersion: development
recordedCodexSchemaDigest: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
instructions: []
```

The two digests above are illustrative, not a usable image/profile. Use the actual immutable engine image ID and the expected build-recorded schema digest. `engineExecutable` must be an exact native absolute path on the Host; it names the engine, never a provider executable. `instructions` is an ordered list of `{boundary, path}` coordinates for already captured source files. Missing instructions request a new complete acquisition and remain incomplete; this adapter never reads instruction paths.

Preparation resolves the selected image ID, copies only `/opt/markitect/runtime.env` through `Engine.ReadImageFile`, and checks the image identity again. The engine creates a container for copying and removes it without starting it. It does not pull/build an image, start a Session, invoke either provider, probe a CLI version or export a schema.

The supported metadata format is exactly four unique LF-separated records emitted by [the runtime Dockerfile](../../../packaging/runtime/Dockerfile): `codex`, `claude`, `codexSchema` and `agentd`. Values must be bounded nonempty UTF-8 literals without control characters or surrounding whitespace. The recorded Codex version must equal `codex-cli 0.162.0`; the recorded agentd version and Codex schema digest must match the selected profile. The Claude record is retained through the whole file identity, with no assertion that this package supports Claude sessions. Unknown/duplicate records, old profile versions, tags and passthrough fields fail closed.

Apply binds the actual immutable image ID, canonical metadata path, whole-file SHA-256 and size, literal CLI/agentd versions and build-recorded Codex schema digest. The schema digest is a build assertion, not an observed schema inventory or provider handshake. File acquisition is bounded to 1 MiB. Engine errors and cancellation preserve completed observations as incomplete; known mismatches are failures. Neither outcome grants runtime readiness or execution authority.

Tests use the real Apply profile consumer with source fixtures and fake image-reader peers. Metadata bytes are synthetic examples of the repository's runtime format; no third-party schema or proprietary CLI source is copied. Concrete factory tests bind regular files without executing them. These tests provide no real engine, provider, Session, recovery or Apply evidence.

## Pinned protocol codec

`NewCodec` supplies pure in-memory encoding and NDJSON framing for a selected stable Codex App Server 0.162.0 subset. It encodes `initialize`, `initialized`, `thread/start`, a single text input in `turn/start` with optional effort and object output schema, and `turn/interrupt`. It does not implement `run.Sessions`, start processes, read credentials, redact secrets, execute tools or correlate outstanding requests. Encoded instructions are literal caller inputs, including explicit empty strings; the codec selects no runtime policy or instruction sources.

`Feed` accepts fragmented or multiple lines. Each line, including LF, is bounded by the caller's limit, at most 1 MiB; JSON nesting is bounded to 64 compound levels. Invalid UTF-8, unpaired Unicode surrogate escapes, duplicate keys at any depth, trailing JSON values and malformed envelopes are refused. This prevents Go's JSON replacement of malformed surrogate escapes from collapsing distinct wire IDs. A malformed or oversized line latches the decoder's failure and returns earlier complete observations plus a rejected observation. The rejected oversized observation is only a bounded prefix and has `Truncated` set. Later bytes in that fragment are not interpreted; the caller must stop consuming the affected connection. `End` reports incomplete final bytes without supplying an invented LF; `Pending` retains a copied unfinished prefix.

`Message.Raw` returns a copy of the original provider NDJSON bytes, including LF, leading/trailing whitespace and CR when present. `Digest` hashes exactly those bytes, or the explicitly truncated prefix. This boundary is not the engine command, agentd frame headers, reconstructed tool envelopes or a redacted journal representation. The codec provides no replay or dispatch authority. A future Host consumer must redact known secrets before every journal append while retaining the separate true wire identity.

Request IDs retain separate signed-integer and string forms. Thread and turn IDs have distinct types and come only from the corresponding provider fields. `ReadResponse` requires the caller's expected request ID and method. It validates the selected identity/status projection and required field presence, not the full upstream schemas: unconsumed configuration, thread fields and turn items remain opaque in `Raw`. No result acceptance follows from a provider's `completed` status. `failed` and `interrupted` remain distinct. Unknown notifications and unsupported server requests have explicit kinds and keep their raw bytes; a consumer must resolve their significance rather than discard them as harmless. Unknown turn statuses and inconsistent selected event shapes are refused with retained raw observations.

The wire shapes were checked against the official [`rust-v0.162.0` commit c1382380de69521303b416720a52f42d51af6248](https://github.com/openai/codex/tree/c1382380de69521303b416720a52f42d51af6248/codex-rs/app-server-protocol/schema/json), not current unversioned examples. Paths in the table are relative to that commit's `codex-rs/app-server-protocol/schema/json/`; hashes identify the complete upstream bytes consulted. These are documentation references, not vendored schema files. The tests contain independently authored synthetic examples, not captured provider traffic or upstream schema copies.

| Upstream file | SHA-256 |
| --- | --- |
| `JSONRPCMessage.json` | `6da4cc373a7ccd01f5608ad4496eeb4d44b6a9b0d3158e85ed66b59f3c0b5114` |
| `RequestId.json` | `9c72d59a306d827113020c6d31a14c326d55210d9bd6954f825196a52009f1c7` |
| `ClientNotification.json` | `706cf248d75027c84a3c63348d0ed507182e8eba40069dd17541793de029145a` |
| `v1/InitializeParams.json` | `b55124e7c0f36a1789396daa5578217a92be67f053903696236686e59f6f212e` |
| `v1/InitializeResponse.json` | `62ad689c2cb6379913c1d72749cfd8de5089d35760214123518eb92eef11acc9` |
| `v2/ThreadStartParams.json` | `e9c6d3cc18d049bfbc0249add3808fb8e5a27e1a0b5a423767aafbc3859a9428` |
| `v2/ThreadStartResponse.json` | `7247cd531d3f78174444f8262fd3090d5acf8ab74084547b4e0a831d74e77c9a` |
| `v2/ThreadStartedNotification.json` | `9d4763337276cf6ff56b6ee611edfaa652801e01737bd5fc461353e12aa86505` |
| `v2/TurnStartParams.json` | `583aff13e7a2da43fbb0150e0edc3c73171f374f0aa745dddba256166988a005` |
| `v2/TurnStartResponse.json` | `20e837d07826b9c01578ff40b962ab08f58a355a9984e96226733498514e9bad` |
| `v2/TurnStartedNotification.json` | `74050381ae8a90db4893fc15fc26ef14560762c3d320c3fa8d9df9b1aba1273b` |
| `v2/TurnCompletedNotification.json` | `69413578ec3ecd02691424bdeb9d640c7244b4ca4c5c3ef26f69a2efe7c37db8` |
| `v2/TurnInterruptParams.json` | `6dff382dae73d1dbc58406ed045605f647e7a49660e2540fbd2c6c24d60c5f2b` |
| `v2/TurnInterruptResponse.json` | `531de6be06fe979b5963f249bab82498a175e614bf65ac12fb2e849dfe60bcf1` |

The upstream root `LICENSE` at the same commit is Apache-2.0, SHA-256 `d17f227e4df5da1600391338865ce0f3055211760a36688f816941d58232d8dc`. No schema generator, CLI installation, schema export, image build or provider invocation is part of this codec. These source tests establish neither a runtime handshake nor recovery, Session readiness or Apply support.
