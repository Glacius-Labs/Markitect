# Development

Markitect's current product workflow is the model-first `markitect <verb>` surface: a committed recursive Manager model under `.markitect/` owns project intent and artifact responsibility. Draft model changes are proposals; commits and digests bind bytes but do not authenticate human approval. Native instructions and scoped execution operate with caller permissions, not an OS sandbox. Retain mechanisms for their current engineering purpose; remove compatibility-only functions. Published releases and original evidence remain immutable historical records.

For coordinated parallel work, follow [Pull requests and parallel work](#pull-requests-and-parallel-work) below and the [roadmap](docs/implementation-plan.md#how-we-work); the [development guide](docs/development/README.md) and [parallel-work flow](docs/development/parallel-work.md) give the details. Shared semantic contracts remain coordinator-owned; the normal gates below apply to every integrated candidate. For documentation placement and ownership, read the [documentation maintenance guide](docs/development/documentation.md).

## Ownership and layout

Current source has four product layers ([Architecture](docs/architecture.md#layers)):

- **Core:** the structural compiler, snapshots and project-model views.
- **Infrastructure:** Git and working-tree access, and guarded writes.
- **Application:** the use cases and their CLI and MCP surfaces.
- **Runtime:** the inner roles in owned candidate workspaces.

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
- Run the cheap gates first: the full project `check`, module and architecture checks, and the documentation check (see [Verify a change](#verify-a-change)). A new source or test file must be covered by its Manager's Artifact path. A change to a module configuration must keep its declared checks synchronized with the implementation.

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
go run ./src/cmd/markitect check --repo .
go run ./src/cmd/markitect-check-modules --repo . --hooks .markitect/modules/githooks.config --pipelines .markitect/modules/pipelines.config
go run ./src/cmd/markitect-check-architecture --repo .
go run ./src/cmd/markitect check --repo . --revision HEAD
go run ./src/cmd/markitect context --repo . --revision HEAD product
$base = git merge-base HEAD origin/main
go run ./src/cmd/markitect impact --repo . --since $base --revision HEAD
python -B scripts/check-docs.py
go test ./... -count=1 -timeout=60m
go vet ./...
go run ./tools/smoke -repo .
git diff --check
```

Tests must not depend on the machine's Git configuration. A package whose tests create repositories or run git, directly or through production code, calls `testkit.Main` from its `TestMain`; it gives the test process a dedicated home directory and global Git configuration and keeps the Go caches. New fixtures use `testkit.NewRepo` for repositories with a fixed identity, fixed dates and unchanged line endings, and `testkit.TempDir` for short temporary paths with retried cleanup.

The runtime process end-to-end scenarios in `projectrun` have the build constraint `!runtime_unit`. Normal `go test ./...` and the required Quality/Race jobs include every scenario. For a focused pass over the remaining tests, use `go test -tags runtime_unit ./src/internal/host/projectrun -count=1 -timeout=30m`; this selection is not a complete source gate. Compare its test list with `go test ./src/internal/host/projectrun -list .` to see the excluded scenarios. The shared repository template uses `testkit.NewRepo`; each scenario supplies a private child descriptor through argv, with its own logs and temporary directory. Only the re-executed child loads those settings. Parallel scenarios never call `t.Setenv` or change the parent working directory. Keep a scenario's descriptor stable while its child invocations run, and wait for those invocations before cleanup.

Full source gates use `-count=1` to require fresh test execution rather than report a cached test success as a new candidate run. The complete suite runs once per CI job. The structural project-model check does not execute its declared Check commands; CI invokes the repository gates explicitly. Focused checks use a ten-minute default, broad package runs thirty minutes and a complete local suite run sixty minutes. Preserve already-running work and original timeout evidence; investigate a real expiry rather than automatically retrying it.

Pull requests and main commits run the hosted quality job on Linux; that job is the required gate. It has a 30-minute limit and gives the suite a 20-minute test-binary limit, so a hung test still ends with Go's timeout report. Windows runs the same job nightly, on manual dispatch and in releases, with a 90-minute job limit and a 60-minute test-binary limit, and does not block merges ([DEC-013](docs/concepts/register.md#dec-013-linux-first-for-tests-and-the-playground)). The job also executes the package, adapter, schema and example gates and the fixed-revision dogfood Verify.

A parallel race-detector job runs the complete untagged suite with `go test -race` on Linux and is part of the pull-request gate. TEST-03 isolates and parallelizes the runtime process scenarios; their inclusion is unchanged. Durations depend on source and runner conditions; preserve the source and scope of any timing comparison. Run `go test -race ./...` yourself (Linux or macOS with cgo) when you change concurrent code.

Go sources must be gofmt-clean: CI fails when `gofmt -l` lists any tracked Go file, so run `gofmt -w` on the files you change.

`go run ./tools/smoke -repo .` packages the source, tests the standalone bootstrap, builds the tools from the packaged source and checks their model-first flows end to end, including preview/write behavior, fixed-revision impact and Verify, and the project-world example. CI runs it in the quality job. `go run ./tools/smoke bundle -revision <commit>` runs the same checks against the release bundle installed into a consumer repository; the release workflow runs it on Linux and Windows with the bundle hash from its preflight.

The [playground smoke](.github/workflows/playground-smoke.yaml) runs the case playground's unit tests and its provider-free container run: a fake agent drives the Markitect arm (`project init`, `onboard`, `setup`, every station and the final `project check`) against a Linux build of the commit. It runs nightly, on manual dispatch and on pull requests that change `src/`, `go.mod`, `go.sum` or `experiments/case-playground/`. It makes no model call and uses no secrets.

CI never retries a test. Each run attempt uploads its Go test results, and the weekly [flake report](.github/workflows/flake-report.yaml) lists the tests that passed and failed on the same commit and OS, plus every failing test, over the last 14 days. Run it on demand from the Actions tab.

For prose-only changes, validate the fixed candidate with `check`, selected `context` and `impact`, managed-artifact accounting, relevant local links/anchors and any changed command examples, plus independent documentation review. Do not repeat an unchanged full suite locally merely to refresh a prose-only SHA; retain its actual tested revision. Required hosted CI and release gates still apply, and focused documentation checks are not a new full Project Verify result.

Run `python -B scripts/check-docs.py` for local links and heading/HTML anchors in the maintained entry documents. Pass exact repository-relative Markdown paths to check additional changed pages, including historical records. Run `python -B -m unittest discover -s scripts -p test_check_docs.py` when changing the checker. CI runs both commands. The [documentation maintenance guide](docs/development/documentation.md) describes scope and placement.

The standalone checks do not require another repository or an AI model. CI runs the Linux gate on every pull request and the Windows job nightly. A successful source gate establishes only the product checks that ran; it does not establish semantic correctness or an adopting project's acceptance. The import checker at `src/internal/tooling/architecture` statically examines production and test imports, including supported-platform files, and has negative fixtures for forbidden directions. `go test ./...` includes the repository architecture test, while CI also runs `markitect-check-architecture` as a named gate. The release quality job reuses CI. These routes are wired; wiring is not a gate result. Consult the [consolidation report](docs/validation/clean-architecture-consolidation.md) for exact-head status, and do not add an exception allowlist.

Edit canonical model YAML and regenerate its readable view with the digest-bound `docs` preview/write workflow in [AGENTS.md](AGENTS.md). Current project operations and release tooling are documented in [Project operations](docs/project-operations.md) and the [distribution guide](integration/README.md). The [roadmap](docs/implementation-plan.md) owns current source status. Earlier Project/Domain commands and content-package contracts remain available in the immutable [v0.14.1 source and guide](https://github.com/Glacius-Labs/Markitect/tree/v0.14.1).

## Historical Project/Domain contracts

The removed Project/Domain command surface, rendering rules and content-package contract are preserved at the immutable [v0.14.1 tag](https://github.com/Glacius-Labs/Markitect/tree/v0.14.1). Current contributors use the model-first [Project workflow](docs/project-workflow.md) and current [operations guide](docs/project-operations.md); no conversion of older project files is implied.

## Release work

Local package, bundle and install preparation uses `go run ./src/cmd/markitect-release COMMAND`; see [current source release tooling](integration/README.md#current-source-release-tooling). Product and release tools share independent build metadata. The legacy fixture benchmark is manual historical tooling; it does not automatically run on model-first published releases.

Markitect is licensed under [Apache-2.0](LICENSE); the root license is included in source distributions. The canonical [third-party notices](src/internal/tooling/licenses/notices.md) are embedded in the CLI and included in both source distribution paths. When dependencies or the build toolchain change, compare their upstream notices and update this file in the same candidate when needed. Check `markitect licenses` from the packaged bootstrap as well as the source build.

[Operations and releases](docs/operations.md) describes the supported source and publication gates. The [roadmap](docs/implementation-plan.md) owns current source status; [GitHub Releases](https://github.com/Glacius-Labs/Markitect/releases) lists available distributions. The [production assessment](docs/production-assessment.md) records dated release evidence. A source version does not imply acceptance by any adopting project.

If a restricted local environment refuses the default Go build cache, use an explicit cache outside snapshot inputs, for example `GOCACHE` at `.cache/go-build` or an external workspace. Do not place a growing cache at an admitted source path and assume `.gitignore` filters working-tree snapshots: Markitect uses its own documented source exclusions. This is development-environment setup, not a change to snapshot semantics or the published CLI's permissions.

## Historical release contracts

The v0.14.1 Project/Domain and canonical Projection contracts are preserved at the immutable [release tag](https://github.com/Glacius-Labs/Markitect/tree/v0.14.1). They are not current source interfaces or contributor gates.
