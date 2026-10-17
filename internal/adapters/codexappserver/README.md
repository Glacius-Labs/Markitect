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
