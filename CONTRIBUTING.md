# Development

## Ownership and layout

| Path | Responsibility |
|---|---|
| `cmd/markitect` | CLI flags, exit codes, and output |
| `internal/core` | Typed resources, project configuration, and dependency graph |
| `internal/format` | Strict YAML parsing and schema generation |
| `internal/source` | Working-tree and immutable Git snapshots |
| `internal/inputs` | Explicit ordinary-file inputs |
| `internal/app` | Context, impact, evidence, verification, rendering, and controlled writes |
| `internal/authoring` | Embedded core authoring resources |
| `internal/render` | Generic managed views and declared output adapters |
| `internal/release` | Deterministic source archives and release bundles |
| `internal/contentpackage` | Validated offline content-package archive format |
| `internal/app/install.go` | Release pin plans and application |
| `integration` | Versioned distribution support files |
| `examples` | Executable synthetic product example |
| `docs` | Product architecture, usage, decisions, and roadmap |

Adopting repositories own their content and any import scripts used to bring existing material into the Markitect model. Markitect does not embed a repository-specific migration or renderer policy. Core authoring guidance remains part of the product.

## Verify a change

From the repository root with Go 1.27.1 or later:

```powershell
go test ./...
go vet ./...
go run ./cmd/markitect schema --repo .
go run ./cmd/markitect check --repo examples/minimal
git diff --check
```

The standalone checks do not require another repository or an AI model. CI runs supported Windows and Linux gates. A successful source gate establishes only the product checks that ran; it does not establish semantic correctness or an adopting project's acceptance.

Edit validation declarations and regenerate schemas with `schema --repo . --write`. Edit example YAML, then run `format`, `render --repo examples/minimal --write`, and `check`. For package or consumer example changes, also run `go run ./cmd/markitect check --repo examples/package-consumer`. Edit core authoring at `internal/authoring/resources/*.yaml`; its content is canonical and embedded in the binary. Tests compile those resources through the ordinary application API. The v0.3.0 content-package contract and consumer workflow are documented in [Content packages](docs/content-packages.md); the executable package fixtures live under `examples/content-package` and `examples/package-consumer`.

## Project checks and rendering

For a Project to produce complete `verify` evidence, declare every required command under `spec.checks`. Each check has a stable `name` and a `run` argument array. The first item must be a bare executable name resolvable through `PATH`; use an interpreter command such as `go run tools/check-docs.go` for a repository script. Markitect passes arguments directly and does not insert a shell. This avoids implicit shell expansion and keeps the executed command visible in the fixed Project snapshot.

Checks run against a materialized fixed revision within the documented time and output bounds. They run with the caller's local authority; the materialized copy is not a sandbox. Do not configure commands that mutate the checkout or access unrelated data. Missing or empty check declarations make `verify` incomplete rather than successful.

Rendering always supports Markitect's generic managed views. Additional output targets and rule adapters must be declared in the Project. Only the declared outputs are rendered or checked. A provider-specific renderer or an import script belongs in the adopting repository that owns its format and policy.

## Release work

[Operations and releases](docs/operations.md) describes the supported source and publication gates. The v0.1.0 release is historical; v0.2.0 is a verified published release. See the [production assessment](docs/production-assessment.md) for exact release evidence and [GitHub Releases](https://github.com/Glacius-Labs/Markitect/releases) for available distributions. A source version does not imply acceptance by any adopting project.
