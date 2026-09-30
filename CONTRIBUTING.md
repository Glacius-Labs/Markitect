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

These checks require neither Konfyra nor the Cockpit. Initial Go toolchain/module download may require network access. Source/bootstrap changes require Windows and Linux coverage before a release. Hosted CI wiring follows the selected repository host; these local commands are its intended gates, not proof that hosted CI ran.

Edit the Go validation declarations and use `schema --repo . --write` to update schemas. Edit example YAML and use `format` and `render --repo examples/minimal --write` on a working branch to update it. The example regression checks all expected resources, its binding, declared file content and generated views.

## Release and consumers

Keep development builds distinguishable from accepted releases. Set a release version only when its required evidence exists. `package --repo . --output NEW_DIRECTORY` creates the tool archive and current flat YAML tool lock. It does not distribute content packages. A release directory must not exist beforehand.

Review source, tests, archive digest, bootstrap and source provenance before updating a consumer. The consumer owns its content, integration changes and acceptance. Moving this checkout does not change an existing consumer's archive or executable. A new binary digest invalidates reuse of evidence recorded by the old binary.

The original Git history was independently cloned, without shared objects or a dependency on the old checkout. Before publishing, choose the remote host, module path, API domain and license. A host-specific CI definition and signed/reproducible release policy can then be added against concrete distribution requirements.
