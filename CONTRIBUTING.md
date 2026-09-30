# Development

## Layout and ownership

| Path | Responsibility |
|---|---|
| `cmd/markitect` | CLI flags, exit codes and output |
| `internal/core` | Typed resources and dependency graph |
| `internal/format` | Strict YAML and schema generation |
| `internal/source` | Working and immutable Git snapshots |
| `internal/inputs` | Explicit ordinary file inputs |
| `internal/app` | Context, impact, evidence and controlled writes |
| `internal/authoring` | Embedded core authoring resources compiled through the application API |
| `internal/render` | Managed Markdown/provider projections |
| `internal/migrate` | Existing Konfyra migration adapter |
| `internal/release` | Deterministic source archives and release bundles |
| `internal/app/install.go` | Verified five-file consumer pin plan and application |
| `integration` | Standard-library Go consumer bootstrap |
| `examples` | Executable, synthetic usage example |
| `docs` | Product architecture, use and decisions |

## Verification

From the repository root, using writable Go caches if needed:

```powershell
go test ./...
go vet ./...
go run ./cmd/markitect schema --repo .
go run ./cmd/markitect check --repo examples/minimal
git diff --check
```

These checks require neither Konfyra nor the Cockpit. Initial Go toolchain/module download may require network access. Source/bootstrap changes require Windows and Linux amd64 coverage before a release. [GitHub CI](.github/workflows/ci.yaml) runs the standalone gates on both systems. The tagged release workflow is designed to repeat bundle install and smoke checks on both before publishing. A workflow definition alone is not evidence: inspect successful runs for the exact candidate commit before using its assets.

Edit the Go validation declarations and use `schema --repo . --write` to update schemas. Edit example YAML and use `format` and `render --repo examples/minimal --write` on a working branch to update it. The example regression checks all expected resources, its binding, declared file content and generated views.

Edit bundled authoring at `internal/authoring/resources/*.yaml`. Its content is canonical and embedded in the binary; do not maintain a second provider-specific copy. Tests parse and compile the bundle through the ordinary pipeline. Source packaging must preserve all required embedded assets. Structural query provenance belongs to core resolution, never a second reference resolver in a CLI adapter. [Measurement](docs/measurement.md) describes the bounded performance and authoring exercises.

## Release and consumers

[Operations and releases](docs/operations.md) defines the supported boundary, mandatory release evidence, recovery and consumer rollback procedure.

The source declares version `0.1.0`. A source version is not proof of a published or accepted release; use the exact immutable release metadata and successful hosted run for the intended source commit as the release evidence. The private repository has immutable GitHub Releases enabled. The current GitHub plan does not permit the private repository's server-side branch-rule change (the API returns 403); CLI rejection of `main` and `master` does not provide equivalent server enforcement.

The supported release workflow uses `bundle --repo . --revision FULL_COMMIT --output ABSENT_ZIP`. Bundling reads the immutable Git snapshot, verifies the source version, and creates one complete five-file pin bundle: a `release.yaml` manifest plus the lock, source archive, bootstrap and paired test. The tag must be `v` plus the source version. CI verifies and installs the same bundle on Windows and Linux amd64; the final Release contains that ZIP, both native binaries, and provenance YAML with the tag, source commit, run, toolchain and payload hashes. The release is assembled as a draft, then published so GitHub can make the tag and assets immutable. The actual tag/run remains the release evidence; a workflow file or local build is not acceptance.

`install --repo CONSUMER --bundle BUNDLE --sha256 VERIFIED_OUTER_HASH` is a read-only preflight by default. Review the plan and manifest version/source commit against the verified release and its provenance. Add `--write` only on a named non-protected branch after committing a Git baseline. Install writes five files atomically one at a time; it is not a repository transaction. Inspect returned `written` paths and recovery guidance if the operation stops partway through. Consumer integration and rollback remain in the consumer's normal Git review route.

`package --repo . --output NEW_DIRECTORY` remains a low-level development command that emits only `tools/markitect/source.zip` and the flat YAML tool lock. It is not the consumer release bundle and includes no bootstrap, manifest, native binary, or provenance. The output directory must not exist beforehand.

Review source, tests, archive digest, bootstrap and source provenance before updating a consumer. The consumer owns its content, integration changes and acceptance. Moving this checkout does not change an existing consumer's archive or executable. A new binary digest invalidates reuse of evidence recorded by the old binary.

The original Git history was independently cloned, without shared objects or a dependency on the retired checkout, and is preserved in the private `Glacius-Labs/Markitect` repository. Development belongs in the Glacius Labs checkout; consumer repositories contain pinned distributions. The Go module follows that repository identity. The API domain placeholder and public distribution license are still undecided; private immutable release attestations are distinct from a claim that public provenance or domain policy is settled.
