# Project operations

The current commands are `check`, `context`, `docs`, `version`, `licenses` and `help`. `apply` reports unavailable before acquisition or effects. Build from `cmd/markitect`; [Project workflow](project-workflow.md) owns the sequence and [CONTRIBUTING](../CONTRIBUTING.md) owns source gates.

## Structural and metadata stages

`check --repo PATH` calls actual PreparationService with an explicit canonical v1 manifest and dependencies. Reports retain each acquisition attempt and actual compiler, execution, registry, file-accounting and context-selection statuses. Failed or incomplete acquisition cannot become an empty passing model.

Metadata and settings are unrequested unless `--metadata` is selected. `--host-profile` names an explicit absolute Orchestrator document; no environment or provider discovery substitutes for it. Agent and Check adapters parse selected executable/schema/profile bytes only. Product defaults come from their embedded versioned source, and effective settings retain provenance. Reports neither start a process nor establish execution readiness.

## Why a Definition is in context

`context <package> --repo PATH` selects one local Package, including intermediate or empty Packages from the compiler inventory; `.` selects the ordinary root. It reports actual Definition content, resolved values, provenance and inclusion reasons. Imported Packages are not local Managers. It creates no task prompt, mandate, run, budget or evidence record.

## CLI, setup and current roots

The default manifest is `.markitect/model.yaml`. Repeated `--dependency` flags select required manifests. Repository-relative coordinates remain relative to the selected repository; absolute manifests establish explicit acquisition boundaries. `--revision` resolves a ref once through the repository port and preserves its full commit. Fixed acquisition never falls back to current external Orchestrator or dependency bytes.

This increment has no MCP, automatic setup, provider session, workspace creation, executable check runner or artifact publication command. The old generation remains in [Git](https://github.com/Glacius-Labs/Markitect/tree/5ff3206917fa3b7482d9daedb903d57a919f56b8). [Provider adapters](provider-adapters.md) describes the metadata boundary.

## Generated view and recovery

`docs` renders actual compiled v1 Definitions to `docs/markitect/project.md`. Its content digest identifies rendered bytes; its preview digest also binds sources and the observed target. Write with `--write --expect PREVIEW_DIGEST` after reviewing that preview. The writer checks the source/target plan under an independent guard, refuses protected inputs and historical writes, reads back actual bytes and retains completed paths after partial failure. Do not infer a filesystem transaction or artifact Apply.

Documentation previews may include explicitly selected external sources. For this increment, `docs --write` requires every acquired model, control and instruction source to have an actual repository binding. Otherwise it returns `not_completed` before effects because target disjointness has not been established. Repository-bound dependencies remain writable; an external source is not silently reclassified or followed through an alternate native spelling.

## Historical operation links

The earlier [MCP groups](https://github.com/Glacius-Labs/Markitect/blob/5ff3206917fa3b7482d9daedb903d57a919f56b8/docs/project-operations.md#mcp-server-and-tool-groups), [Work Item workflow](https://github.com/Glacius-Labs/Markitect/blob/5ff3206917fa3b7482d9daedb903d57a919f56b8/docs/project-operations.md#ordinary-work-item) and [candidate recovery](https://github.com/Glacius-Labs/Markitect/blob/5ff3206917fa3b7482d9daedb903d57a919f56b8/docs/project-operations.md#candidate-workspaces-and-recovery) retain their original scope.
