# Project operations

The current commands are `check`, `context`, `docs`, `apply`, `extract`, `setup`, `version`, `licenses` and `help`. Build from `cmd/markitect`; [Project workflow](project-workflow.md) owns the sequence and [CONTRIBUTING](../CONTRIBUTING.md) owns source gates.

The later [command design](design/canonical-model/commands.md) records the accepted Verify, Extract, Optimize and other operation semantics. Except for the first bounded `extract` below, it does not add them to this executable surface or authorize their execution.

## Structural and metadata stages

`check --repo PATH` calls actual PreparationService with an explicit canonical v1 manifest and dependencies. Reports retain each acquisition attempt and actual Model check, execution, registry, file-accounting and context-selection statuses. Failed or incomplete acquisition cannot become an empty passing model.

Metadata and settings are unrequested unless `--metadata` is selected. `--runtime-profile` names an explicit absolute Runtime profile document; no environment or provider discovery substitutes for it. Agent and Check adapters parse selected executable/schema/profile bytes only. Product defaults come from their embedded versioned source, and effective settings retain provenance. Reports neither start a process nor establish execution readiness.

## Why a Definition is in context

`context <package> --repo PATH` selects one local Package, including intermediate or empty Packages from the Model check inventory; `.` selects the ordinary root. It reports actual Definition content, resolved values, provenance and inclusion reasons. Imported Packages are not local Managers. It creates no task prompt, mandate, run, budget or evidence record.

## CLI, setup and current roots

The default manifest is `.markitect/model.yaml`. Repeated `--dependency` flags select required manifests. Repository-relative coordinates remain relative to the selected repository; absolute manifests establish explicit acquisition boundaries. `--revision` resolves a ref once through the repository port and preserves its full commit. Fixed acquisition never falls back to current external Runtime profile or dependency bytes.

The CLI has no MCP server and no automatic setup. The old generation remains in [Git](https://github.com/Glacius-Labs/Markitect/tree/5ff3206917fa3b7482d9daedb903d57a919f56b8). [Provider adapters](provider-adapters.md) describes the metadata and runtime boundary.

## Apply

`apply` needs a committed repository, an explicit absolute Runtime profile (`--runtime-profile`) and an explicit absolute run configuration (`--run-config`); without them it reports `not_completed` before acquisition. It starts real Agents in containers, runs the required checks, and publishes only a verified candidate; it never commits. Its JSON report names the evidence directory under the run configuration's state directory. A Run is closed: no question or answer crosses Root to the user or the invoking AI, there is no `answer` command and the run configuration accepts no owner decisions. A model that lacks a material decision or contradicts itself ends the run Failed without publication ([DEC-063](concepts/register.md#dec-063-the-model-is-authoritative-defects-escalate), [closed Runs](design/canonical-model/manager-apply-workflow.md#mandatory-model-assessment-and-closed-runs)); amend the model and start a new run. The [runtime contract](design/canonical-model/manager-apply-runtime.md) owns behavior, evidence and limits; the [workflow](project-workflow.md#execute-verify-and-apply) gives the sequence.

## Generated view and recovery

`docs` renders actual checked v1 Definitions to `docs/markitect/project.md`. Its content digest identifies rendered bytes; its preview digest also binds sources and the observed target. Write with `--write --expect PREVIEW_DIGEST` after reviewing that preview. The writer checks the source/target plan under an independent guard, refuses protected inputs and historical writes, reads back actual bytes and retains completed paths after partial failure. Do not infer a filesystem transaction or artifact Apply.

Documentation previews may include explicitly selected external sources. For this increment, `docs --write` requires every acquired model, control and instruction source to have an actual repository binding. Otherwise it returns `not_completed` before effects because target disjointness has not been established. Repository-bound dependencies remain writable; an external source is not silently reclassified or followed through an alternate native spelling.

## Historical operation links

The earlier [MCP groups](https://github.com/Glacius-Labs/Markitect/blob/5ff3206917fa3b7482d9daedb903d57a919f56b8/docs/project-operations.md#mcp-server-and-tool-groups), [Work Item workflow](https://github.com/Glacius-Labs/Markitect/blob/5ff3206917fa3b7482d9daedb903d57a919f56b8/docs/project-operations.md#ordinary-work-item) and [candidate recovery](https://github.com/Glacius-Labs/Markitect/blob/5ff3206917fa3b7482d9daedb903d57a919f56b8/docs/project-operations.md#candidate-workspaces-and-recovery) retain their original scope.
