# Development

Markitect's current product workflow is the model-first `markitect project` surface: a committed recursive Manager model under `.markitect/` owns project intent and artifact responsibility. Draft model changes are proposals; commits and digests bind bytes but do not authenticate human approval. Native instructions and scoped execution operate with caller permissions, not an OS sandbox. Retain mechanisms for their current engineering purpose; remove compatibility-only functions. Published releases and original evidence remain immutable historical records.

For coordinated parallel work, follow [Pull requests and parallel work](#pull-requests-and-parallel-work) below and the [roadmap](docs/implementation-plan.md#how-we-work); the [development guide](docs/development/README.md) and [parallel-work flow](docs/development/parallel-work.md) give the details. Shared semantic contracts remain coordinator-owned; the normal gates below apply to every integrated candidate. For documentation placement and ownership, read the [documentation maintenance guide](docs/development/documentation.md).

## Ownership and layout

Current source has four product layers and the legacy line ([Architecture](docs/architecture.md#layers)):

- **Core:** the structural compiler, snapshots and project-model views.
- **Infrastructure:** Git and working-tree access, and guarded writes.
- **Application:** the use cases and their CLI and MCP surfaces.
- **Runtime:** the inner roles in owned candidate workspaces.
- **Legacy:** the earlier Project/Domain line, which ARCH-09 removes.

Outside the layers:

- `src/internal/testkit` holds hermetic test fixtures: Git isolation, repositories with fixed identity and dates, and short temporary directories. Only test files import it, and it imports no other Markitect package; the import gate enforces both.
- `integration` holds the standalone public bootstrap that is copied into installed packages. It imports no Markitect package.
- `packaging/winget` holds versioned portable package manifests derived from verified releases.

The [code map](docs/development/code-map.md) lists every package with its layer, purpose and owning document. Read that document before you change a package. [Modules and static composition](docs/development/modules.md) owns the import rules. Executables under `src/cmd` import Host only and delegate to it. Fixture repositories and reference data live under `examples/`; they are not Go packages. The [vision](docs/vision.md) owns the product thesis; implementation and gate evidence belong to the [roadmap](docs/implementation-plan.md) and source-bound validation reports.

Snapshot semantics and the boundary between generic values and Git operations are documented in [Source snapshots](docs/source-snapshots.md). Keep Git resolution and process hardening in `src/internal/infrastructure/source`; keep deterministic comparison over resolved values in `src/internal/core/snapshot`. Repository branch, index, and worktree checks belong to the write use cases that require them. Do not add alternate production providers or a provider framework without a concrete consumer.

Adopting repositories own their content and any import scripts used to bring existing material into the Markitect model. Markitect does not embed a repository-specific migration or renderer policy.

The product treats adopting-project files as [declared artifact inputs](docs/architecture.md#project-artifact-boundary), including source code. Its own Go implementation and release tooling do not imply a source-code analysis feature for adopting projects.

Keep source and test files focused on one coherent responsibility. When new functionality introduces an independent responsibility, prefer a new file in the existing package. Keep related types and helpers together; do not add packages or abstractions solely to shorten files.

## Pull requests and parallel work

Several sessions change this repository at the same time, each in its own worktree. The [roadmap](docs/implementation-plan.md#how-we-work) owns sessions, work packages, [zones](docs/implementation-plan.md#zones) and the [backlog](docs/work-items/backlog.yaml). These rules keep the work from colliding.

**Branches and pull requests**

- Start every branch from current `origin/main`. Give it a meaningful lower-case name with a prefix: `dev/`, `fix/`, `docs/` or `exp/`. No tool name, date, version or number ([branch names](docs/implementation-plan.md#branch-names)).
- Open one pull request per work package. Its title starts with the package ID, for example `ARCH-04: Remove the canonical projection alpha`.
- Change only files in your package's zone. An edit outside it needs the integrator's approval; say so in the pull request.
- Put status, acceptance evidence, findings and open owner decisions in the pull-request description. Sessions do not edit the backlog, the roadmap or the concept register; the integrator does.

**Before every push**

- Run `git show --stat` for each new commit, or `git diff --name-only origin/main HEAD`, and check that only the intended files changed.
- Never use `git stash`. The stash is shared by all worktrees of this repository, so a pop can apply another session's changes. Use a patch file instead (`git diff > change.patch`, later `git apply change.patch`) or a separate worktree.
- Run the cheap gates first: managed-artifact accounting, `check`, `format`, the module checks and the documentation check (see [Verify a change](#verify-a-change)). A new file under a managed root needs an owner entry in `markitect-artifacts.yaml`. A change to `.github/workflows/ci.yaml` needs a renewed digest in `.markitect/modules/pipelines.config`.

**While CI runs**

- The pull-request head is frozen while the required Linux CI runs. Nobody pushes to it during that run.
- A review fix may be pushed while only a run that is not required (Windows) is still going ([DEC-013](docs/concepts/register.md#dec-013-linux-first-for-tests-and-the-playground)).
- Push fixes as new commits. Do not force-push a pull request that is under review.

**Merging**

- The integrator merges in dependency order, one pull request at a time, with merge commits, once Linux CI is green.
- Before each merge, the integrator compiles the pull request merged onto current main (`go vet ./...`). Two pull requests that pass alone can still break main together.
- A stacked pull request, whose base is another branch, must be retargeted to `main` before it merges. A pull request merged into a branch that has already merged never reaches main.
- To bring a branch up to date, merge `origin/main` into it or rebase unpushed commits. Do not rewrite commits that are already pushed.

## Verify a change

Two rules shape every change:

- Keep examples executable. Every example is run by a test or by one of the checks below.
- Scope tests to concrete risks and required gates. A test checks Markitect itself, not provider CLI details, permissions, line endings or other environment problems ([DEC-016](docs/concepts/register.md#dec-016-a-clean-stable-testable-main-and-uniform-structure-first)).

Begin through the repository's model-first contributor guidance and classify a requested change as model intent, implementation or both. For an intent change, edit its canonical owner, review the generated readable document, and commit the accepted model before implementation. A passing check, digest or provider report is not human approval. From the repository root with Go 1.27.1 or later and Git 2.40 or later (Explore, Plan and Apply call `git check-attr --source`):

```powershell
go test ./... -count=1 -timeout=60m
go vet ./...
go run ./src/cmd/markitect schema --repo .
go run ./src/cmd/markitect check --repo examples/minimal
go run ./src/cmd/markitect check --repo examples/repository-layout
go run ./src/cmd/markitect check --repo examples/canonical-engineering
go run ./src/cmd/markitect format --repo examples/canonical-engineering
go run ./src/cmd/markitect model --repo examples/canonical-engineering
go run ./src/cmd/markitect check --repo examples/engineering-constitution
go run ./src/cmd/markitect format --repo examples/engineering-constitution
go run ./src/cmd/markitect model --repo examples/engineering-constitution
go run ./src/cmd/markitect check --repo examples/engineering-discovery
go run ./src/cmd/markitect format --repo examples/engineering-discovery
go run ./src/cmd/markitect check --repo examples/software-architecture
go run ./src/cmd/markitect format --repo examples/software-architecture
go run ./src/cmd/markitect model --repo examples/software-architecture
go run ./src/cmd/markitect context --repo examples/software-architecture --namespace engineering --kind Skill --name implement-order
go run ./src/cmd/markitect check --repo examples/delivery-target-equality
go run ./src/cmd/markitect format --repo examples/delivery-target-equality
go run ./src/cmd/markitect model --repo examples/delivery-target-equality
go run ./src/cmd/markitect context --repo examples/delivery-target-equality --namespace engineering --kind Skill --name deployment-review
go run ./src/cmd/markitect check --repo benchmark/fixtures/v2
git diff --check
```

Tests must not depend on the machine's Git configuration. A package whose tests create repositories or run git, directly or through production code, calls `testkit.Main` from its `TestMain`; it gives the test process a dedicated home directory and global Git configuration and keeps the Go caches. New fixtures use `testkit.NewRepo` for repositories with a fixed identity, fixed dates and unchanged line endings, and `testkit.TempDir` for short temporary paths with retried cleanup.

Full source gates use `-count=1` to require fresh test execution rather than report a cached test success as a new candidate run. The complete suite runs once per CI job; this source Project's `verify` does not declare it as a check, so the dogfood Verify does not run it a second time. Focused checks use a ten-minute default, broad package runs thirty minutes and a complete local suite run sixty minutes. Preserve already-running work and original timeout evidence; investigate a real expiry rather than automatically retrying it.

Pull requests and main commits run the hosted quality job on Linux; that job is the required gate. It has a 30-minute limit and gives the suite a 20-minute test-binary limit, so a hung test still ends with Go's timeout report. Windows runs the same job nightly, on manual dispatch and in releases, with a 90-minute job limit and a 60-minute test-binary limit, and does not block merges ([DEC-013](docs/concepts/register.md#dec-013-linux-first-for-tests-and-the-playground)). The job also executes the package, adapter, schema and example gates and the fixed-revision dogfood Verify.

The [playground smoke](.github/workflows/playground-smoke.yaml) runs the case playground's unit tests and its provider-free container run: a fake agent drives the Markitect arm (`project init`, `onboard`, `setup`, every station and the final `project check`) against a Linux build of the commit. It runs nightly, on manual dispatch and on pull requests that change `src/`, `go.mod`, `go.sum` or `experiments/case-playground/`. It makes no model call and uses no secrets.

CI never retries a test. Each run attempt uploads its Go test results, and the weekly [flake report](.github/workflows/flake-report.yaml) lists the tests that passed and failed on the same commit and OS, plus every failing test, over the last 14 days. Run it on demand from the Actions tab.

For prose-only changes, validate the fixed candidate with `check`, selected `context` and `impact`, managed-artifact accounting, relevant local links/anchors and any changed command examples, plus independent documentation review. Do not repeat an unchanged full suite locally merely to refresh a prose-only SHA; retain its actual tested revision. Required hosted CI and release gates still apply, and focused documentation checks are not a new full Project Verify result.

Run `python -B scripts/check-docs.py` for local links and heading/HTML anchors in the maintained entry documents. Pass exact repository-relative Markdown paths to check additional changed pages, including historical records. Run `python -B -m unittest discover -s scripts -p test_check_docs.py` when changing the checker. CI runs both commands. The [documentation maintenance guide](docs/development/documentation.md) describes scope and placement; this repository check is separate from the optional product documentation-router feature.

The standalone checks do not require another repository or an AI model. CI runs the Linux gate on every pull request and the Windows job nightly. A successful source gate establishes only the product checks that ran; it does not establish semantic correctness or an adopting project's acceptance. The import checker at `src/internal/tooling/architecture` statically examines production and test imports, including supported-platform files, and has negative fixtures for forbidden directions. The repository test calls it through `go test ./...`, CI has a named gate step, the explicit Project check delegates through Host, and the release quality job reuses CI. These routes are wired; wiring is not a gate result. Consult the [consolidation report](docs/validation/clean-architecture-consolidation.md) for exact-head status, and do not add an exception allowlist.

Edit validation declarations and regenerate schemas with `schema --repo . --write`. Edit example YAML, then run `format`, `render --repo examples/minimal --write`, and `check`. For package or consumer example changes, also run `go run ./src/cmd/markitect check --repo examples/package-consumer`. Edit core authoring at `src/internal/host/embedded/resources/*.yaml`; its content is canonical for that embedded guidance. Tests compile those resources through the ordinary application API. The model-first project path is specified in [Project workflow](docs/project-workflow.md); compiler commands and immutable release histories are described in [Usage](docs/usage.md#legacy-projectdomain-cli-compatibility). The [roadmap](docs/implementation-plan.md) owns current source status. The v0.3.0 content-package contract and consumer workflow are documented in [Content packages](docs/content-packages.md); executable fixtures live under `examples/content-package` and `examples/package-consumer`.

## Legacy Project/Domain checks and rendering

These rules apply to the legacy line, which still runs this repository's own checks until ARCH-07. New projects use the model-first [Project workflow](docs/project-workflow.md).

For a Project to produce complete `verify` evidence, declare every required command under `spec.checks`. Each check has a stable `name` and a `run` argument array. The first item must be a bare executable name resolvable through `PATH`; use an interpreter command such as `go run tools/check-docs.go` for a repository script. Markitect passes arguments directly and does not insert a shell. This avoids implicit shell expansion and keeps the executed command visible in the fixed Project snapshot.

Checks run against a materialized fixed revision within the documented time and output bounds. They run with the caller's local authority; the materialized copy is not a sandbox. Do not configure commands that mutate the checkout or access unrelated data. Missing or empty check declarations make `verify` incomplete rather than successful.

Rendering writes only explicitly selected outputs. Add `markdown` to `spec.targets` to generate resource views under `docs/markitect/`; add Codex and Claude targets, rule mappings and shared entrypoints for the provider outputs the Project owns. Provider entrypoints link directly to canonical YAML. Only declared outputs are rendered or checked. An adopting repository owns its root navigation, hooks, custom output formats and import scripts.

## Published top-level `init` behavior

The published v0.14.1 release has a top-level `markitect init` for the legacy line; current source has removed it. New model-first repositories use `markitect project init`, which creates the `.markitect/project.yaml` model tree, runtime, ignore rules and generated readable view. Use the command supported by the selected source or release; do not reinterpret existing Project files as project-model sources or silently convert them.

## Release work

Markitect is licensed under [Apache-2.0](LICENSE); the root license is included in source distributions. The canonical [third-party notices](src/internal/tooling/licenses/notices.md) are embedded in the CLI and included in both source distribution paths. When dependencies or the build toolchain change, compare their upstream notices and update this file in the same candidate when needed. Check `markitect licenses` from the packaged bootstrap as well as the source build.

[Operations and releases](docs/operations.md) describes the supported source and publication gates. The [roadmap](docs/implementation-plan.md) owns current source status; [GitHub Releases](https://github.com/Glacius-Labs/Markitect/releases) lists available distributions. The [production assessment](docs/production-assessment.md) records dated release evidence. A source version does not imply acceptance by any adopting project.

If a restricted local environment refuses the default Go build cache, use an explicit cache outside snapshot inputs, for example `GOCACHE` at `.cache/go-build` or an external workspace. Do not place a growing cache at an admitted source path and assume `.gitignore` filters working-tree snapshots: Markitect uses its own documented source exclusions. This is development-environment setup, not a change to snapshot semantics or the published CLI's permissions.

## Canonical Projection alpha (removed)

The experimental canonical Projection alpha bundled with v0.14.1 has been removed from current source; its contracts remain documented at the [v0.14.1 tag](https://github.com/Glacius-Labs/Markitect/blob/v0.14.1/docs/canonical-projections.md). `src/internal/core` remains the structural compiler for Schema, Kind, Property and Definition that the model-first project model uses. The v0.13 Domain and policy kernel and its consumers stay in current source under `src/internal/host/compat/v0_13` until ARCH-09 removes them; neither is an authoring format for model-first projects. Do not silently translate their resources into `.markitect/model/`.
