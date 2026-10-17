# Project workflow

Markitect's purpose is canonical model-first development with delegated realization. This source increment implements read-only preparation and a guarded generated-model view. Recursive work, sessions, executable checks, candidate workspaces, publication and Apply remain unavailable; the [delivery plan](implementation-plan.md) describes their implementation sequence.

## Connect and inspect

Use Git with an existing commit and the root-layout CLI:

```text
go run ./cmd/markitect check --repo .
go run ./cmd/markitect context product/model --repo .
```

The default manifest is `.markitect/model.yaml`. `--model` selects another explicit manifest. Supply every required dependency manifest with repeated `--dependency PATH`; there is no discovery from folders, registries or old project formats. The manifest and `model-lock.yaml` must satisfy the [canonical v1 contract](design/canonical-model/minimal-language/language-contract.md).

An omitted revision observes the actual working draft, including Git-ignored files. `--revision REF` resolves once and captures exactly that full commit. Neither preview supplies an Apply publication guard. A fixed preview refuses present-day external inputs without historical revision binding, including an external Host document even when REF is HEAD.

## Model and readiness

`check` requests structural preparation by default: actual compilation, execution declarations, registry, complete file accounting and static context selection. `context <package>` uses the same preparation and reports selected Definitions, provenance and inclusion reasons for one local Package, or `.` for the ordinary root.

Add `--metadata --host-profile ABSOLUTE_DOCUMENT_PATH` to request static Agent/Check profile metadata and settings preparation. Actual adapters parse explicitly selected source-bound executable, schema and profile files. A metadata pass does not establish runtime support. Unrequested metadata and settings report `requested: false`; absent or incomplete required observations remain incomplete.

## Generated documentation

```text
go run ./cmd/markitect docs --repo .
go run ./cmd/markitect docs --repo . --expect PREVIEW_DIGEST --write
```

Read the nested `view` in the preview. `canonicalModelDigest` identifies compiled model meaning; `contentDigest` hashes rendered Markdown bytes; `previewDigest` binds observed sources, target bytes/existence/mode and generated content. Pass only the returned `previewDigest` to `--expect`. Changed sources or targets refuse the old plan. Fixed-revision views never write, and protected model/control/instruction inputs cannot become generated targets. A write uses an independent documentation guard and actual read-back, with completed paths retained if a later check fails. It is separate from artifact Apply.

Documentation previews may include explicitly selected external sources. For this increment, `docs --write` requires every acquired model, control and instruction source to have an actual repository binding. Otherwise it returns `not_completed` before effects because target disjointness has not been established. Repository-bound dependencies remain writable; an external source is not silently reclassified or followed through an alternate native spelling.

## Execute, verify, and apply

`apply` returns `not_completed` and exit 3 before acquiring a source or performing an effect. No preview, digest, test result or metadata report makes it available. This CLI has no execution lifecycle or MCP server. The [design contracts](design/canonical-model/README.md) and assigned package dispatches govern later stages.

## Existing repositories

Author the intended canonical v1 model and explicit dependencies, then register exact realization files in `.markitect/registry/<package>/registry.yaml`. File granularity is the author's choice; a file may realize several Definitions but has one explicit owner. An exceptional ignore names one exact observed file with a reason. Missing entries, missing paths, duplicate ownership and overlaps with protected inputs fail. Gitignore is not coverage.

There is no automatic conversion, scaffold or Brownfield runtime. Earlier workflows remain reproducible from their [immutable source](https://github.com/Glacius-Labs/Markitect/tree/5ff3206917fa3b7482d9daedb903d57a919f56b8).

## Record setup and interrupted work

These operations create no run, task, budget, candidate or recovery journal. Preserve acquisition diagnostics and actual generated-document completed-path reports. Execution/recovery records belong to real ports and implementation, not invented preview records.

## Acceptance and status

JSON reports name the operation, requested stage and actual statuses. Exit 0 means the requested stage passed; 1 is a known failure; 2 is invalid usage; 3 is incomplete observation, I/O/budget refusal or unavailable operation. `failed` and `not_completed` remain distinct. A committed model binds a specification revision; it does not authenticate a human decision. Local source checks, hosted CI, provider execution, releases and owner acceptance remain separate evidence.
