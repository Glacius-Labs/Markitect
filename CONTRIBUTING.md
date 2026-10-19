# Development

Markitect is being rebuilt around canonical language v1 and delegated realization. Current source provides actual read-only preparation, a guarded generated model view and the recursive Apply of [DEC-056](docs/concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace) on branch `dev/recursive-apply-simplified`, pending review, CI and owner acceptance ([runtime contract](docs/design/canonical-model/manager-apply-runtime.md)). [AGENTS.md](AGENTS.md), the [delivery plan](docs/implementation-plan.md) and the bounded dispatch own authority and shared-file reservations.

## Ownership and layout

All maintained Go code belongs to one root module. `cmd/markitect` delegates to `internal/bootstrap`; pure model, snapshot, organization and realization values have no infrastructure dependency. `internal/apply` owns preparation and consumer ports. Concrete adapters implement repository capture, static profile metadata and document writes. Independent maintenance services have explicit `internal/tooling/` responsibilities; that parent grants no general import exemption.

The [code map](docs/development/code-map.md), [architecture](docs/architecture.md) and [dependency rules](docs/development/modules.md) describe actual packages. Shared hermetic `internal/testkit` is test-only and imports no Markitect package. The standalone bootstrap in `integration` and source smoke in `tools/smoke` import no product package.

The self-model is `.markitect/model.yaml`, its `model-lock.yaml` and Definition tree. An exact registry entry assigns each artifact to one responsibility and actual Definition basis. Canonical/control inputs are protected separately. An exact ignore needs a reason and never supplies missing realization. Implementation-only changes need not alter intent.

## Verify a change

Use Git and Go from `go.mod`. Run meaningful local checks for the actual change; the commands below also define the repository-wide required CI gates. Follow the [delivery plan](docs/implementation-plan.md#integration) for background CI and direct delivery merges. Do not repeat unrelated full suites for unchanged source. Do not edit sources while a gate runs, and await each running Go process before another Go command in the same worktree.

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

Before integration, obtain independent complete-diff review and resolve findings. Freeze the exact head and await its required CI in the background. The delivering chat merges directly with a merge commit and `--match-head-commit`; recompose only for conflicts or substantive overlap. Do not monitor Main CI or retrieve artifacts after each merge; respond to reported Main failures. Local checks do not replace required CI. Publication, provider runs, studies and owner acceptance need separate authority and evidence.

## Historical Project/Domain contracts

The retired v0.14.1 distribution and its [preserved source and guide](https://github.com/Glacius-Labs/Markitect/tree/v0.14.1) retain their contract. The preceding source generation remains at [5ff32069](https://github.com/Glacius-Labs/Markitect/tree/5ff3206917fa3b7482d9daedb903d57a919f56b8). Current consumers do not reinterpret those formats or revive their runtime.

## Release work

[Operations](docs/operations.md) and [distribution](integration/README.md) own separate release work. Maintenance entrypoints are under `cmd/markitect-release`, `cmd/markitect-check-architecture` and `cmd/markitect-check-modules`. Source packages retain the root license, embedded [notices](internal/tooling/licenses/notices.md), strategy defaults and required test data. Use an external build cache when needed; Gitignore does not filter the complete product census. A source version does not establish a published or accepted release.
