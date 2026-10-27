# Development

Markitect is built around canonical language v1 and delegated realization. Current source provides read-only preparation, the guarded generated model view, standalone hierarchical Runs and Apply through stored Verify approval and guarded publication. Extract, Cockpit and interactive authoring are retired. [AGENTS.md](AGENTS.md), the [delivery plan](docs/implementation-plan.md) and the bounded dispatch own authority and shared-file reservations. Source availability does not establish native runtime qualification. Markitect itself is developed as an ordinary software project without a canonical self-model ([DEC-076](docs/concepts/register.md#dec-076-markitect-is-developed-as-a-regular-software-project)).

## Ownership and layout

All maintained Go code belongs to one root module. `cmd/markitect` delegates to `internal/bootstrap`; pure model, snapshot, organization and realization values have no infrastructure dependency. `internal/project` owns structural preparation and candidate validation. `internal/run` owns closed hierarchical execution; `internal/runtimeconfig` owns static native/check metadata. Concrete adapters implement repository capture, static profile metadata and document writes; `internal/guardedwrite` guards every working-tree write. Independent maintenance services have explicit `internal/tooling/` responsibilities; that parent grants no general import exemption.

One rule table in [`internal/tooling/architecture/rules.go`](internal/tooling/architecture/rules.go) gives every package its layer and allowed imports; the [architecture](docs/architecture.md#layers) explains it and the generated [code map](docs/development/code-map.md) lists every package. Shared hermetic `internal/testkit` is test-only and imports no Markitect package. The standalone bootstrap in `packaging/bootstrap` and source smoke in `tools/smoke` import no product package.

Intent lives in ordinary documentation, one canonical page per topic ([documentation maintenance](docs/development/documentation.md)); a change updates that page together with the code. Target projects keep `.markitect/`; product code and tests that handle it use fixtures, never this repository's own files.

## Verify a change

Use Git and Go from `go.mod`. Run meaningful local checks for the actual change. During development, automatic PR, Main-push and scheduled CI are paused; full CI remains available by manual dispatch and to the release workflow. Independent review and focused local checks are the default merge basis under the [delivery plan](docs/implementation-plan.md#integration). Do not repeat unrelated full suites for unchanged source. Do not edit sources while a gate runs, and await each running Go process before another Go command in the same worktree.

For prose-only changes, check links in the changed pages with `python -B scripts/check-docs.py PATH...` and run `git diff --check`. For code or control changes, run the relevant tests and validators; broaden validation when the changed contract or a concrete failure calls for it. Record actual results and limitations in the PR.

The full pipeline runs Linux quality checks and the race detector in about five minutes. CI has no Windows job.

```text
gh workflow run ci.yaml --ref BRANCH
```

Manual CI is optional for ordinary development merges. When explicitly required for an assignment, assess its result on the exact reviewed head and resolve relevant failures before integration. The Flake report is also manual. Release-owned calls retain their full validation. The commands below remain available for local or full validation:

```text
go mod verify
go test ./... -count=1 -timeout=20m
go vet ./...
go build ./...
go test ./internal/tooling/architecture
python -B -m unittest discover -s scripts -p test_*.py
python -B scripts/check-docs.py
python -B -m unittest discover -s tools/library-oracle
python -B -m unittest discover -s tools/comparison
python -B -m unittest discover -s tools/makerspace-oracle -p test_*.py
python -B -m unittest discover -s tools/model-analysis
```

Run broad Go suites on Linux, for example in WSL: on Windows, real Git is much slower, and `internal/adapters/gitrepo` alone takes 23 to 40 minutes instead of about two. Tests that do not test time take their Run deadline from the test's own deadline, so the `-timeout` value bounds a hang.

Run `gofmt` on changed Go files and `git diff --check`. CI checks module checksums, tests with and without the race detector, vet, build, tracked Go formatting, every production/test/platform import against the architecture rule table, the packaged-source smoke, repository scripts, maintained documentation links, the independent Library oracle with its bounded transport, and the unit tests of the comparison, Makerspace oracle and model analysis tools. Obsolete engines and their tests/gates are removed together; no skips or success stubs may replace required checks.

Regenerate the code map when the rule table or a package doc comment changes:

```text
go test ./internal/tooling/architecture -run TestCodeMapPage -update
```

Run the packaged source smoke in a fresh external work directory:

```text
go run ./tools/smoke -repo . -work ABSOLUTE_EMPTY_EXTERNAL_DIRECTORY
```

It packages and builds root-layout source, tests the standalone bootstrap and drives actual Library M1/M2 preparation, static context, fixed-source refusals and Apply's rejection of missing or legacy invocation configuration before resource acquisition. It starts no Agent or container, so it does not run an Apply, implement the Library or establish provider/Apply/release acceptance. The `bundle` smoke belongs to separately authorized release work and writes an isolated consumer installation.

## Pull requests and parallel work

The [delivery plan](docs/implementation-plan.md) owns current scopes, dependencies, path reservations and integration order, and the [working method](docs/development/working-method.md) governs design learning and owner orientation. This section owns execution inside an assigned package. Candidate progress and checks live in the package's chat and PR; there is no separate backlog or dashboard.

### Assignment

Read [AGENTS.md](AGENTS.md), the plan and the technical contract. Verify checkout, branch, full SHA and dirty state before edits. Use the coordinator's exact integrated base and one assigned non-protected branch and isolated worktree per writer. Preserve unrelated work; never stash across worktrees, reset another checkout or force-push reviewed heads.

One chat owns one bounded package or API slice and its tests. The dispatch lists paths, inputs, acceptance and dependency gates. Subagents are permitted within that scope. Unresolved shared semantics block dependent implementation; independent work can proceed. No model or provider claim replaces actual verification.

### Shared files and contracts

Shared contracts and dependencies need explicit coordination. The coordinator owns root dependencies and the plan; the plan's [work streams](docs/implementation-plan.md#work-streams) assign exclusive scopes, and everything else stays with the coordinator. Others request a concrete change, with intended behavior, consumer and cases, rather than editing a shared file. Two chats cannot own the same contract, generated output or helper.

Producers own value types; consumers own ports. Add no global types or interface package and no duplicate definitions to unblock an adapter; [Architecture](docs/architecture.md#adding-changing-or-removing-a-package) states how packages are added. Tests and fixtures live with their package. An external dependency needs the coordinator's agreement and an exact pin; root module changes have one writer.

### Integration

Run focused checks and inspect the actual changed files; preserve failures and unknowns with their exact source identity. Open a focused PR with the package, scope, contracts, base and head and the validation, and distinguish implemented behavior from scaffolding or proposed APIs. Regenerate generated views from their sources.

Before integration, obtain independent complete-diff review, run relevant local checks and resolve findings. Freeze the exact reviewed head. The delivering chat merges directly with a merge commit and `--match-head-commit`, without waiting for hosted CI by default, as the plan's [integration rules](docs/implementation-plan.md#integration) state; recompose only for conflicts or substantive overlap and renew relevant review/checks after a changed candidate. If an assignment explicitly requires manual CI, bind it to that head and wait in the background. Preserve failed observations and distinguish local from hosted evidence. A source package is not a real Agent run, a successful Library Apply, a release or human acceptance: publication, provider runs, studies and owner acceptance need separate scope, finite authority and evidence.

## Release work

[Operations](docs/guides/operations.md) and [distribution](packaging/README.md) own separate release work. The maintenance entrypoint is `cmd/markitect-release`. Source packages retain the root license, embedded [notices](internal/buildinfo/notices.md), required test data and explicit invocation examples. Use an external build cache when needed; Gitignore does not filter the complete product census. A source version does not establish a published or accepted release.
