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
| `internal/release` | Deterministic tool source archive |
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

These checks require neither Konfyra nor the Cockpit. Initial Go toolchain/module download may require network access. Source/bootstrap changes require Windows and Linux coverage before a release. [GitHub CI](.github/workflows/ci.yaml) runs the standalone gates on Windows and Linux. A workflow definition alone is not proof of a successful hosted run; inspect the candidate's actual run before provisioning its artifact.

Edit the Go validation declarations and use `schema --repo . --write` to update schemas. Edit example YAML and use `format` and `render --repo examples/minimal --write` on a working branch to update it. The example regression checks all expected resources, its binding, declared file content and generated views.

Edit bundled authoring at `internal/authoring/resources/*.yaml`. Its content is canonical and embedded in the binary; do not maintain a second provider-specific copy. Tests parse and compile the bundle through the ordinary pipeline. Source packaging must preserve all required embedded assets. Structural query provenance belongs to core resolution, never a second reference resolver in a CLI adapter. [Measurement](docs/measurement.md) describes the bounded performance and authoring exercises.

## Release and consumers

Keep development builds distinguishable from accepted releases. Set a release version only when its required evidence exists. `package --repo . --output NEW_DIRECTORY` creates the tool archive and current flat YAML tool lock. It does not distribute content packages. A release directory must not exist beforehand.

Review source, tests, archive digest, bootstrap and source provenance before updating a consumer. The consumer owns its content, integration changes and acceptance. Moving this checkout does not change an existing consumer's archive or executable. A new binary digest invalidates reuse of evidence recorded by the old binary.

The original Git history was independently cloned, without shared objects or a dependency on the retired checkout, and is preserved in the private `Glacius-Labs/Markitect` repository. Development belongs in the Glacius Labs checkout; consumer repositories contain pinned distributions. The Go module follows that repository identity. Before a public release, choose the controlled API domain, license and release provenance/signing policy. The current CI source artifact is a development distribution with an exact digest, not a production acceptance or signed release.
