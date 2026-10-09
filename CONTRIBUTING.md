# Development

Markitect's current product workflow is the model-first `markitect project` surface: a committed recursive Manager model under `.markitect/` owns project intent and artifact responsibility. Draft model changes are proposals; commits and digests bind bytes but do not authenticate human approval. Native instructions and scoped execution operate with caller permissions, not an OS sandbox. Retain mechanisms for their current engineering purpose; remove compatibility-only functions. Published releases and original evidence remain immutable historical records.

For coordinated parallel work, read the [development guide](docs/development/README.md), [parallel-work flow](docs/development/parallel-work.md) and your [workstream specification](docs/workstreams/README.md). Shared semantic contracts remain coordinator-owned; the normal gates below apply to every integrated candidate.

## Ownership and layout

| Path | Final responsibility |
|---|---|
| `cmd/...` | Thin executable entrypoints. Every CLI imports Host only and delegates to Host runtime functions. |
| `internal/modules/projectmodel`, `internal/host/projectwork`, `internal/host/projectcli` | Model-first project compiler, repository snapshots/coverage, Manager workflow and the `project` command surface. The model declares semantics and file responsibilities; it does not infer source-language meaning. |
| `internal/host/projectrun`, `internal/host/projectadoption`, `internal/host/projectbriefing`, `internal/host/projectonboarding` | Bounded execution, selected Brownfield proposals, committed-model briefs and native contributor guidance. These do not authenticate human approval or provide OS isolation. |
| internal/core | Experimental canonical-alpha structural compiler for Schema, Kind, Property and Definition with pure normalized IR; no legacy policy, authoring source, activation, provider or execution semantics. |
| internal/host | Host frontend and composition; installable manifest activation; compatibility use cases; explicit Core input selection; model/context/impact; request construction, process/check execution, operational persistence and controlled writes. |
| internal/modules/adoption | New selected capture/review implementation helpers; Go package only, not an installable Module manifest. |
| internal/host/compat/v0_13/consumers | Isolated historical v0.13.0 consumers with unchanged compatibility semantics; new capability packages remain separate. |
| internal/host/compat/v0_13/consumers/agentrules | Historical v0.13 Codex/Claude consumers; compatibility only. |
| internal/modules/markdown | Experimental canonical-alpha Markdown capability implementation; not itself an installable manifest. |
| internal/host/compat/v0_13/consumers/artifactcoverage | Historical v0.13 artifact accounting consumer; compatibility only. |
| internal/host/records | Pure operational ProjectionRecord, VerificationResult and ownership-index validation/encoding; Host owns filesystem persistence, execution and safe writes. |
| internal/host/compat/v0_13/consumers/githooks and consumers/pipelines | Historical v0.13 bounded artifact checks; compatibility only. |
| internal/modules/dotnet | Experimental canonical-alpha explicitly mapped .NET capability implementation; not itself an installable manifest. |
| internal/host/compat/v0_13/consumers/github and consumers/azuredevops | Historical v0.13 offline consumers; compatibility only. |
| `internal/infrastructure/source` | Git/working-tree acquisition, process hardening and materialization into snapshot values. |
| `internal/tooling/architecture`, `internal/tooling/release`, `internal/tooling/publish`, `internal/tooling/licenses` | Mechanical import gate, immutable distribution/publication operations, and canonical notices. |
| `integration` | Standalone public bootstrap/distribution Tooling; copied into installed packages, with no Markitect package imports. |
| `examples`, `benchmark`, `experiments` | Executable fixtures and measurements; Harness, not production dependencies. |
| `packaging/winget` | Versioned portable package manifests derived from verified releases. |
| `docs` | Product architecture, usage, decisions and canonical roadmap. |

The [architecture overview](docs/architecture.md#go-ownership-and-final-dependency-model) and [Module guide](docs/development/modules.md) define these boundaries. The published v0.14.1 Project/Domain CLI and canonical Projection alpha remain versioned compatibility contracts. The [vision](docs/vision.md) owns the model-first product thesis; exact implementation and gate evidence belong to the [roadmap](docs/implementation-plan.md) and source-bound validation reports.

Snapshot semantics and the boundary between generic values and Git operations are documented in [Source snapshots](docs/source-snapshots.md). Keep Git resolution and process hardening in `internal/infrastructure/source`; keep deterministic comparison over resolved values in `internal/core/snapshot`. Repository branch, index, and worktree checks belong to the write use cases that require them. Do not add alternate production providers or a provider framework without a concrete consumer.

Adopting repositories own their content and any import scripts used to bring existing material into the Markitect model. Markitect does not embed a repository-specific migration or renderer policy. Core authoring guidance remains part of the product.

The product treats adopting-project files as [declared artifact inputs](docs/architecture.md#project-artifact-boundary), including source code. Its own Go implementation and release tooling do not imply a source-code analysis feature for adopting projects.

Keep source and test files focused on one coherent responsibility. When new functionality introduces an independent responsibility, prefer a new file in the existing package. Keep related types and helpers together; do not add packages or abstractions solely to shorten files.

## Verify a change

Begin through the repository's model-first contributor guidance and classify a requested change as model intent, implementation or both. For an intent change, edit its canonical owner, review the generated readable document, and commit the accepted model before implementation. A passing check, digest or provider report is not human approval. From the repository root with Go 1.27.1 or later:

```powershell
go test ./... -count=1 -timeout=60m
go vet ./...
go run ./cmd/markitect schema --repo .
go run ./cmd/markitect check --repo examples/minimal
go run ./cmd/markitect check --repo examples/repository-layout
go run ./cmd/markitect check --repo examples/canonical-engineering
go run ./cmd/markitect format --repo examples/canonical-engineering
go run ./cmd/markitect model --repo examples/canonical-engineering
go run ./cmd/markitect check --repo examples/engineering-constitution
go run ./cmd/markitect format --repo examples/engineering-constitution
go run ./cmd/markitect model --repo examples/engineering-constitution
go run ./cmd/markitect check --repo examples/engineering-discovery
go run ./cmd/markitect format --repo examples/engineering-discovery
go run ./cmd/markitect check --repo examples/software-architecture
go run ./cmd/markitect format --repo examples/software-architecture
go run ./cmd/markitect model --repo examples/software-architecture
go run ./cmd/markitect context --repo examples/software-architecture --namespace engineering --kind Skill --name implement-order
go run ./cmd/markitect check --repo examples/delivery-target-equality
go run ./cmd/markitect format --repo examples/delivery-target-equality
go run ./cmd/markitect model --repo examples/delivery-target-equality
go run ./cmd/markitect context --repo examples/delivery-target-equality --namespace engineering --kind Skill --name deployment-review
go run ./cmd/markitect check --repo benchmark/fixtures/v2
git diff --check
```

Full source gates use `-count=1` to require fresh test execution rather than report a cached test success as a new candidate run. This source Project's `go-tests` check declares a 3600-second command limit and the explicit Go test-binary limit `-timeout=60m`; other repository checks keep their 600-second default.

The hosted quality job has a finite 90-minute limit because it runs the full source suite and then the independent fixed-revision dogfood Verify, whose declared checks include a fresh full suite. The individual test-command limits stay at 60 minutes; the job also executes the package, adapter, schema and example gates.

For prose-only changes, validate the fixed candidate with `check`, selected `context` and `impact`, managed-artifact accounting, relevant local links/anchors and any changed command examples, plus independent documentation review. Do not repeat an unchanged full suite locally merely to refresh a prose-only SHA; retain its actual tested revision. Required hosted CI and release gates still apply, and focused documentation checks are not a new full Project Verify result.

The standalone checks do not require another repository or an AI model. CI runs supported Windows and Linux gates. A successful source gate establishes only the product checks that ran; it does not establish semantic correctness or an adopting project's acceptance. The import checker at `internal/tooling/architecture` statically examines production and test imports, including supported-platform files, and has negative fixtures for forbidden directions. The repository test calls it through `go test ./...`, CI has a named gate step, the explicit Project check delegates through Host, and the release quality job reuses CI. These routes are wired; wiring is not a gate result. Consult the [consolidation report](docs/validation/clean-architecture-consolidation.md) for exact-head status, and do not add an exception allowlist.

Edit validation declarations and regenerate schemas with `schema --repo . --write`. Edit example YAML, then run `format`, `render --repo examples/minimal --write`, and `check`. For package or consumer example changes, also run `go run ./cmd/markitect check --repo examples/package-consumer`. Edit core authoring at `internal/host/embedded/resources/*.yaml`; its content is canonical for that embedded guidance. Tests compile those resources through the ordinary application API. The model-first project path is specified in [Project workflow](docs/project-workflow.md); compiler commands and immutable release histories are described in [Usage](docs/usage.md#legacy-projectdomain-cli-compatibility). The [roadmap](docs/implementation-plan.md) owns current source status. The v0.3.0 content-package contract and consumer workflow are documented in [Content packages](docs/content-packages.md); executable fixtures live under `examples/content-package` and `examples/package-consumer`.

## Legacy Project/Domain checks and rendering

This section preserves the published contract for existing repositories. New projects use the model-first [Project workflow](docs/project-workflow.md).

For a Project to produce complete `verify` evidence, declare every required command under `spec.checks`. Each check has a stable `name` and a `run` argument array. The first item must be a bare executable name resolvable through `PATH`; use an interpreter command such as `go run tools/check-docs.go` for a repository script. Markitect passes arguments directly and does not insert a shell. This avoids implicit shell expansion and keeps the executed command visible in the fixed Project snapshot.

Checks run against a materialized fixed revision within the documented time and output bounds. They run with the caller's local authority; the materialized copy is not a sandbox. Do not configure commands that mutate the checkout or access unrelated data. Missing or empty check declarations make `verify` incomplete rather than successful.

Rendering writes only explicitly selected outputs. Add `markdown` to `spec.targets` to generate resource views under `docs/markitect/`; add Codex and Claude targets, rule mappings and shared entrypoints for the provider outputs the Project owns. Provider entrypoints link directly to canonical YAML. Only declared outputs are rendered or checked. An adopting repository owns its root navigation, hooks, custom output formats and import scripts.

## Legacy top-level `init` behavior

Top-level `markitect init` retains its Project/Domain meaning and layout. New model-first repositories use `markitect project init`, which creates the `.markitect/project.yaml` model tree, runtime, ignore rules and generated readable view. Keep both interfaces explicit; do not reinterpret existing Project files as project-model sources or silently convert them.

## Release work

Markitect is licensed under [Apache-2.0](LICENSE); the root license is included in source distributions. The canonical [third-party notices](internal/tooling/licenses/notices.md) are embedded in the CLI and included in both source distribution paths. When dependencies or the build toolchain change, compare their upstream notices and update this file in the same candidate when needed. Check `markitect licenses` from the packaged bootstrap as well as the source build.

[Operations and releases](docs/operations.md) describes the supported source and publication gates. The [roadmap](docs/implementation-plan.md) owns current source status; [GitHub Releases](https://github.com/Glacius-Labs/Markitect/releases) lists available distributions. The [production assessment](docs/production-assessment.md) records dated release evidence. A source version does not imply acceptance by any adopting project.

If a restricted local environment refuses the default Go build cache, use an explicit cache outside snapshot inputs, for example `GOCACHE` at `.cache/go-build` or an external workspace. Do not place a growing cache at an admitted source path and assume `.gitignore` filters working-tree snapshots: Markitect uses its own documented source exclusions. This is development-environment setup, not a change to snapshot semantics or the published CLI's permissions.

## Canonical Projection compatibility boundary

The experimental canonical Projection alpha bundled with v0.14.1 includes a minimal internal/core structural compiler for Schema, Kind, Property and Definition with pure normalized IR. The historical v0.13.0 Domain/policy kernel and its consumers remain supported behind Host compatibility; neither is the canonical authoring format for new model-first projects. Do not report old contracts as deleted or silently translate their resources into `.markitect/model/`.

Installable extension Modules have exactly one type: schema-only or projection-only. Go capability packages under internal/modules are implementation units, not installable Module manifests. Canonical Projection intent selects spec.representation (initially dotnet or markdown), exact scope, target and policies; it does not hard-bind an implementation. Runtime source configuration projectionBindings pairs the full Projection identity in projection with the exact installed Module name in module, which resolves through the installed version pin and one unique entrypoint. A compatible binding change leaves canonical semantic digest unchanged but stales request-bound plans and records. Reconcile planning follows canonical delta → impact → affected scopes/work proposals; it does not ask users to enumerate implementation files. Preserve fixed-input, stale-plan, exact-path, reviewed-candidate and guarded-apply protections. Conflicts stop before execution, and unknown scope broadens review rather than becoming a no-op.

Operational ProjectionRecords and separate VerificationResults record materialization and bounded check facts; neither is canonical intent or proof of authorization, verifier independence, semantic sufficiency, whole-repository coverage or human acceptance. The reset design and [shared contracts](docs/development/shared-contracts.md#canonical-reset-candidate-contracts) own their experimental-alpha contracts; current source changes require their own validation and do not update the published binary. Do not make release or adopter-acceptance claims from a passing local protocol test.
