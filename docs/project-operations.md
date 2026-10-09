# Managed project operations in the source candidate

The managed `project` workflow described here is an unreleased source capability. It is not part of the published v0.14.1 CLI. Use a matching Markitect source checkout for these commands; the stable release and its Project/Domain contract remain unchanged. The [managed project workflow](project-workflow.md) covers setup and guarded Apply; this page records the current operation and evidence boundaries.

## Repository census and readable documentation

New projects default to `coverageMode: full` and generate the readable model document at `docs/markitect/project.md`. The destination is selected in `.markitect/project.yaml` as `documentPath`. Existing manifests that omit it keep the legacy `.markitect/views/project.md` destination until explicitly changed.

Full mode observes the repository file universe and classifies each in-scope path as a model source, declared realization, Markitect-owned tool file, or an explicit ignore. `.markitect/ignore.yaml` uses a closed `RepositoryIgnore` schema with exact repository-relative file paths or trailing-slash directory prefixes and a required reason:

```yaml
apiVersion: project.markitect.example.org/repository-ignore/v1alpha1
kind: RepositoryIgnore
entries:
  - path: generated/vendor/
    reason: Produced by the external dependency build and reviewed separately
```

Patterns, negation, overlapping entries, `.git` metadata, and `.markitect/` are not accepted as ordinary ignore entries. An ignored path still has an explicit visible classification; ignore entries cannot remove an active required realization. `coverage` returns the `Accounted` and `Conforming` results plus path classifications. `check`, `plan`, and full verification consume the selected coverage contract; a complete census does not prove that source semantics match the model.

`project document --repo .` renders the configured readable view to standard output. Use `--write` to update the project-owned destination from the current model. Generated documentation has one owner and is not a substitute for separately owned project documents.

In full-coverage runs, the Host regenerates this configured document from the integrated final candidate model, includes the resulting bytes in that candidate, and checks that the same final snapshot contains the exact generated document. Missing or stale generated content prevents closure. This keeps the readable view synchronized with the model actually being verified and applied.

## Full verification and whole-model operations

With `coverageMode: full`, a run-based `verify` includes the full Manager assessment in addition to its declared checks. A standalone fixed-revision verification needs no prior implementation run:

```powershell
markitect project verify --repo . --revision COMMIT --write
```

It assesses every active Manager bottom-up against one immutable repository snapshot, supplies parent Managers with their public child contracts and integration results, and runs all declared checks against that same snapshot. Mandatory rules and checks remain in force at every strictness level. The report separates machine checks and Manager evidence; it does not establish semantic truth or human acceptance.

`cleanup` and `reconcile` create plans for the whole Manager tree. Cleanup may improve implementations while preserving the accepted model. Reconcile seeks missing or divergent realizations against the whole model, including areas absent from the original change impact. Both allow a reasoned no-op for a conforming area. They create a plan only; the existing `run`, `verify`, and guarded `apply` operations perform and close the work:

```powershell
markitect project reconcile --repo . --goal "Bring every Manager realization into line with the accepted model" --write
markitect project run --repo . --plan PLAN_ID --write
markitect project verify --repo . --run RUN_ID --write
```

Cleanup uses the same sequence with `project cleanup`. If the desired behavior or authority must change, edit and accept the canonical model separately first. A technical remapping with unchanged semantics must also be an explicit model edit. Neither operation silently broadens or rewrites the accepted world model.

Strictness is configured in `.markitect/runtime.yaml`. It can request additional evidence and counterexamples globally and add Manager-specific requirements. Manager profiles combine with the global defaults; they do not lower mandatory rules, omit required checks, or change the accepted model.

## Model-change briefs

An accepted model change can be recorded against exact committed before/after revisions. Preview the deterministic brief first, then persist it against the current briefing-store digest:

```powershell
markitect project brief --repo . --since BASE_COMMIT --revision MODEL_COMMIT --provenance DECISION_REFERENCE
markitect project brief --repo . --since BASE_COMMIT --revision MODEL_COMMIT --provenance DECISION_REFERENCE --expect STATE_DIGEST --write
markitect project briefings --repo . --manager MANAGER_ID
```

The bundle identifies changed declarations, impact, provenance supplied by the caller, and Manager-specific briefings with public neighbor contracts. It compares the model; it does not infer meaning from prose. The provenance reference is a caller declaration, not authenticated identity. Dismissing an event changes its visibility only; it does not resolve a question, remove a Manager obligation, or convert failed evidence to a pass. Full-coverage model-edit plans reject draft model changes; accept and commit the model separately, then record a brief before assigning work. Plan-bound scoped briefs are included in Manager, Reviewer, and full-Verify contexts, and fresh bindings plus Apply reject changed briefing history. An existing briefing store with a new model digest blocks work until the new change is briefed. If the store is empty and the caller omits `--since`, the Host cannot infer the accepted transition automatically; bootstrap discovery and mandatory creation of the first brief remain explicit workflow gaps.

## Contributor onboarding

After project initialization, preview native instructions for the contributors' chosen agents. The write repeats the same preview and requires its exact digest:

```powershell
markitect project onboard --repo . --provider both
markitect project onboard --repo . --provider both --expect PLAN_DIGEST --write
```

Onboarding installs a shared `.markitect/workflows/model-first.md`, managed instruction blocks in `AGENTS.md` and/or `CLAUDE.md`, and provider-native skill files under `.agents/skills/` and/or `.claude/skills/`. Existing content outside the marked blocks is preserved; malformed or conflicting managed blocks fail before writing. The configured document destination is included in the workflow. Onboarding does not change global agent settings, read credentials, authenticate providers, or start a provider.

Native instructions route agents toward conversational Explore and the model/impact/plan/run/verify/apply workflow. They are ordinary repository files; an agent or contributor with write access can bypass them. Guarded Apply and configured repository checks enforce only their declared boundaries and do not provide an operating-system sandbox.

## Implementation and evidence status

The [9 October validation record](validation/project-operations-2026-10-09.md) binds local gates and the public Shop smoke to their actual source revisions, including the retained failed first suite and its fixture correction.

The source includes deterministic subprocess fixtures for the integrated Run/Verify/guarded-Apply lifecycle; those fixtures test Host protocol mechanics, not real Codex or Claude provider runs (NOT RUN), semantic correctness, complete product acceptance, or productivity gains. Focused onboarding Go tests pass for initialization on a named unborn feature branch, preview and guarded Apply without a commit, a second idempotent preview/write, preservation of pre-existing native instruction bytes, and onboarding the legacy hidden documentation destination without changing its manifest. These test Host writes, not live native providers. Explore drafts, durable open-decision/readiness state, and iterative Brownfield return to the model are still planned. Those gaps remain visible in the [model-first design](design/project-world/model-first-user-workflow.md) and [implementation plan](design/project-world/implementation-plan.md).
