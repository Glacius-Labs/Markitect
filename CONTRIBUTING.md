# Development

## Ownership and layout

| Path | Responsibility |
|---|---|
| `cmd/markitect` | CLI flags, exit codes, and output |
| `cmd/markitect-adapter-dotnet` | External reference adapter for explicitly mapped, captured project-reference declarations |
| `cmd/markitect-release`, `internal/publish` | Maintainer release verification, publication, and distribution metadata |
| `internal/core` | Language registry, typed resources, structural constraints, and relation-specific graphs |
| `internal/snapshot` | Resolved snapshot values, stable content digests, and deterministic snapshot comparison |
| `internal/format` | Strict YAML parsing and schema generation |
| `internal/source` | Git revision and working-tree acquisition, Git process hardening, and source materialization |
| `internal/inputs` | Explicit ordinary-file inputs |
| `internal/app` | Normalized model, context, impact, adapter orchestration, evidence, and controlled writes |
| `internal/authoring` | Embedded core authoring resources |
| `internal/licenses` | Canonical embedded upstream notices |
| `internal/markdownlinks` | Bounded prose destination scanning with source-byte preservation |
| `internal/render` | Opt-in Markdown views and declared output adapters |
| `internal/release` | Deterministic source archives and release bundles |
| `internal/contentpackage` | Validated offline content-package archive format |
| `internal/app/install*.go` | Release pin planning, preflight, and application |
| `integration` | Project adoption and pinned-release guidance |
| `packaging/winget` | Versioned portable package manifests derived from verified releases |
| `examples` | Executable synthetic product example |
| `docs` | Product architecture, usage, decisions, and canonical roadmap |

Snapshot semantics and the boundary between generic values and Git operations are documented in [Source snapshots](docs/source-snapshots.md). Keep Git resolution and process hardening in `internal/source`; keep deterministic comparison over resolved values in `internal/snapshot`. Repository branch, index, and worktree checks belong to the write use cases that require them. Do not add alternate production providers or a provider framework without a concrete consumer.

Adopting repositories own their content and any import scripts used to bring existing material into the Markitect model. Markitect does not embed a repository-specific migration or renderer policy. Core authoring guidance remains part of the product.

The product treats adopting-project files as [declared artifact inputs](docs/architecture.md#project-artifact-boundary), including source code. Its own Go implementation and release tooling do not imply a source-code analysis feature for adopting projects.

Keep source and test files focused on one coherent responsibility. When new functionality introduces an independent responsibility, prefer a new file in the existing package. Keep related types and helpers together; do not add packages or abstractions solely to shorten files.

## Verify a change

From the repository root with Go 1.27.1 or later:

```powershell
go test ./...
go vet ./...
go run ./cmd/markitect schema --repo .
go run ./cmd/markitect check --repo examples/minimal
go run ./cmd/markitect check --repo examples/repository-layout
go run ./cmd/markitect check --repo examples/canonical-engineering
go run ./cmd/markitect format --repo examples/canonical-engineering
go run ./cmd/markitect model --repo examples/canonical-engineering
go run ./cmd/markitect check --repo benchmark/fixtures/v2
git diff --check
```

The standalone checks do not require another repository or an AI model. CI runs supported Windows and Linux gates. A successful source gate establishes only the product checks that ran; it does not establish semantic correctness or an adopting project's acceptance.

Edit validation declarations and regenerate schemas with `schema --repo . --write`. Edit example YAML, then run `format`, `render --repo examples/minimal --write`, and `check`. For package or consumer example changes, also run `go run ./cmd/markitect check --repo examples/package-consumer`. Edit core authoring at `internal/authoring/resources/*.yaml`; its content is canonical and embedded in the binary. Tests compile those resources through the ordinary application API. Project initialization is specified in [Usage](docs/usage.md) and current source status belongs to the [roadmap](docs/implementation-plan.md). The v0.3.0 content-package contract and consumer workflow are documented in [Content packages](docs/content-packages.md); the executable package fixtures live under `examples/content-package` and `examples/package-consumer`.

## Project checks and rendering

For a Project to produce complete `verify` evidence, declare every required command under `spec.checks`. Each check has a stable `name` and a `run` argument array. The first item must be a bare executable name resolvable through `PATH`; use an interpreter command such as `go run tools/check-docs.go` for a repository script. Markitect passes arguments directly and does not insert a shell. This avoids implicit shell expansion and keeps the executed command visible in the fixed Project snapshot.

Checks run against a materialized fixed revision within the documented time and output bounds. They run with the caller's local authority; the materialized copy is not a sandbox. Do not configure commands that mutate the checkout or access unrelated data. Missing or empty check declarations make `verify` incomplete rather than successful.

Rendering writes only explicitly selected outputs. Add `markdown` to `spec.targets` to generate resource views under `docs/markitect/`; add Codex and Claude targets, rule mappings and shared entrypoints for the provider outputs the Project owns. Provider entrypoints link directly to canonical YAML. Only declared outputs are rendered or checked. An adopting repository owns its root navigation, hooks, custom output formats and import scripts.

## Initialization behavior

`init` is a core authoring path for repositories with no Markitect Project. Omitting `--path` selects `.markitect/areas/<namespace>`; an explicit supported path overrides it. Existing configured Areas remain authoritative. See [Repository layout](docs/repository-layout.md) for the convention and compatibility boundaries. Its preview must remain read-only and show exact planned YAML and paths. Write recomputes and validates the plan, then exclusively creates only `markitect.yaml` and one area README under the shared write lock. It must enforce the named non-protected branch boundary and report created paths if a later file creation fails. Keep the two-file operation explicitly non-transactional. An empty Project is structurally checkable but cannot produce complete `verify` evidence until its owner declares actual checks and commits the candidate. Do not add generated policies, resources, checks, targets, package pins, custom templates, or edits to existing root files.

## Release work

Markitect is licensed under [Apache-2.0](LICENSE); the root license is included in source distributions. The canonical [third-party notices](internal/licenses/notices.md) are embedded in the CLI and included in both source distribution paths. When dependencies or the build toolchain change, compare their upstream notices and update this file in the same candidate when needed. Check `markitect licenses` from the packaged bootstrap as well as the source build.

[Operations and releases](docs/operations.md) describes the supported source and publication gates. The [roadmap](docs/implementation-plan.md) owns current source status; [GitHub Releases](https://github.com/Glacius-Labs/Markitect/releases) lists available distributions. The [production assessment](docs/production-assessment.md) records dated release evidence. A source version does not imply acceptance by any adopting project.
