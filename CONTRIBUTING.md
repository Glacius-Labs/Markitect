# Development

Markitect is built around canonical language v1 and delegated realization. Current source retains read-only preparation, the guarded generated model view and a complete standalone hierarchical Run during the approved V1 rebuild; public Apply is explicitly unavailable until S3. Extract, Cockpit and interactive authoring are retired. [AGENTS.md](AGENTS.md), the [delivery plan](docs/implementation-plan.md) and the bounded dispatch own authority and shared-file reservations. Source availability does not establish native runtime qualification.

## Ownership and layout

All maintained Go code belongs to one root module. `cmd/markitect` delegates to `internal/bootstrap`; pure model, snapshot, organization and realization values have no infrastructure dependency. `internal/project` owns structural preparation and candidate validation. `internal/run` owns closed hierarchical execution; `internal/runtimeconfig` owns static native/check metadata. Concrete adapters implement repository capture, static profile metadata and document writes. Independent maintenance services have explicit `internal/tooling/` responsibilities; that parent grants no general import exemption.

The [code map](docs/development/code-map.md), [architecture](docs/architecture.md) and [dependency rules](docs/development/modules.md) describe actual packages. Shared hermetic `internal/testkit` is test-only and imports no Markitect package. The standalone bootstrap in `integration` and source smoke in `tools/smoke` import no product package.

The self-model is `.markitect/model.yaml`, its `model-lock.yaml` and Definition tree. An exact registry entry assigns each artifact to one responsibility and actual Definition basis. Canonical/control inputs are protected separately. An exact ignore needs a reason and never supplies missing realization. Implementation-only changes need not alter intent.

## Verify a change

Use Git and Go from `go.mod`. Run meaningful local checks for the actual change. During development, automatic PR, Main-push and scheduled CI are paused; full CI remains available by manual dispatch and to the release workflow. Independent review and focused local checks are the default merge basis under the [delivery plan](docs/implementation-plan.md#integration). Do not repeat unrelated full suites for unchanged source. Do not edit sources while a gate runs, and await each running Go process before another Go command in the same worktree.

For prose-only changes, check links in the changed pages with `python -B scripts/check-docs.py PATH...` and run `git diff --check`. Keep structural model/accounting preparation where required by [AGENTS.md](AGENTS.md). For code or control changes, run the relevant tests and validators; broaden validation when the changed contract or a concrete failure calls for it. Record actual results and limitations in the PR.

The full pipeline retains Linux quality checks, the race detector and optional Windows validation. Request it explicitly when useful:

```text
gh workflow run ci.yaml --ref BRANCH -f windows=true
```

Manual CI is optional for ordinary development merges. When explicitly required for an assignment, assess its result on the exact reviewed head and resolve relevant failures before integration. The Flake report is also manual. Release-owned calls retain their full validation. The commands below remain available for local or full validation:

```text
go mod verify
go test ./... -count=1 -timeout=20m
go vet ./...
go build ./...
go run ./cmd/markitect-check-architecture --repo .
go run ./cmd/markitect check --repo .
go run ./cmd/markitect check --repo . --revision HEAD
go run ./cmd/markitect-check-modules --repo . --hooks .markitect/modules/githooks.config --pipelines .markitect/modules/pipelines.config
python -B -m unittest discover -s scripts -p test_*.py
python -B scripts/check-docs.py
python -B -m unittest discover -s experiments/library-playground
```

Run `gofmt` on changed Go files and `git diff --check`. CI checks tracked Go formatting, every production/test/platform import, actual package-map entries, complete working/fixed accounting, selected hook/pipeline pins and generated view freshness. The independent Library oracle and bounded transport checks retain their scope. Obsolete engines and their tests/gates are removed together; no skips or success stubs may replace required checks.

Check official generated view bytes:

```powershell
$view = (go run ./cmd/markitect docs --repo . | ConvertFrom-Json).view
if ($LASTEXITCODE -ne 0) { throw 'Preview failed' }
$actual = 'sha256:' + (Get-FileHash -LiteralPath $view.path -Algorithm SHA256).Hash.ToLowerInvariant()
if ($actual -ne $view.contentDigest) { throw 'Generated model view is stale' }
```

After model changes, use the preview's `previewDigest` for the guarded write in [AGENTS.md](AGENTS.md). Regenerate the code map officially when packages or their mapped guidance change:

```text
go test ./internal/tooling/architecture -run TestCodeMapPage -update
```

Run the packaged source smoke in a fresh external work directory:

```text
go run ./tools/smoke -repo . -work ABSOLUTE_EMPTY_EXTERNAL_DIRECTORY
```

It packages and builds root-layout source, tests the standalone bootstrap and drives actual Library M1/M2 preparation, static context, fixed-source refusals and Apply's refusal of a missing or outdated invocation configuration before any acquisition. It starts no Agent or container, so it does not run an Apply, implement the Library or establish provider/Apply/release acceptance. The `bundle` smoke belongs to separately authorized release work and writes an isolated consumer installation.

## Pull requests and parallel work

Verify checkout, branch, full SHA and dirty state before edits. Use the coordinator's exact integrated base and one assigned non-protected branch/worktree per writer. Preserve unrelated work; never stash across worktrees, reset it or force-push reviewed heads. Shared contracts and dependencies need explicit coordination.

Before integration, obtain independent complete-diff review, run relevant local checks and resolve findings. Freeze the exact reviewed head. The delivering chat merges directly with a merge commit and `--match-head-commit`, without waiting for hosted CI by default; recompose only for conflicts or substantive overlap and renew relevant review/checks after a changed candidate. If an assignment explicitly requires manual CI, bind it to that head and wait in the background. Preserve failed observations and distinguish local from hosted evidence. Publication, provider runs, studies and owner acceptance need separate authority and evidence.

## Historical Project/Domain contracts

The retired v0.14.1 distribution and its [preserved source and guide](https://github.com/Glacius-Labs/Markitect/tree/v0.14.1) retain their contract. The preceding source generation remains at [5ff32069](https://github.com/Glacius-Labs/Markitect/tree/5ff3206917fa3b7482d9daedb903d57a919f56b8). Current consumers do not reinterpret those formats or revive their runtime.

## Release work

[Operations](docs/operations.md) and [distribution](integration/README.md) own separate release work. Maintenance entrypoints are under `cmd/markitect-release`, `cmd/markitect-check-architecture` and `cmd/markitect-check-modules`. Source packages retain the root license, embedded [notices](internal/tooling/licenses/notices.md), required test data and explicit invocation examples. Use an external build cache when needed; Gitignore does not filter the complete product census. A source version does not establish a published or accepted release.
