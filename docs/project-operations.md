# Managed project operations in the source candidate

The managed `project` workflow described here is an unreleased source capability. It is not part of the published v0.14.1 CLI. Use a matching Markitect source checkout for these commands; the stable release and its Project/Domain contract remain unchanged. The [managed project workflow](project-workflow.md) covers setup and guarded Apply; this page records the available source operations and separates them from pending final local gates and provider/human evidence.

## Explore, readiness and composed delivery

The source CLI persists a named-scope Explore record, computes fixed-basis readiness, and can record a structure acknowledgement bound to the exact repository/model/runtime/Manager/artifact/check selection. These commands do not start an agent. An acknowledgement stores caller-supplied actor, authority, decision reference and RFC3339 time; it does not authenticate a human. New projects use `workflowMode: guided` and `acceptancePolicy: committed-model`; the committed canonical model is the accepted repository specification for that revision, while a commit or digest does not prove human review.

```text
markitect project explore --repo . --input RECORD.json [--expect PLAN_DIGEST --write]
markitect project readiness --repo . --exploration ID --scope ID
markitect project readiness --repo . --exploration ID --scope ID --acknowledge-structure --actor ACTOR --authority TEXT --decision-ref REF --acknowledged-at RFC3339 --expect DIGEST --write
markitect project deliver --repo . --exploration ID --scope ID --write
markitect project deliver --repo . --exploration ID --scope ID --run RUN_ID --write
```

`deliver` requires an explicitly ready scope, creates one bound plan and advances that durable run through Manager execution, integration, full Verify, Apply preflight and guarded Apply. Passing its run ID resumes the same proof; it does not create a replacement plan. Preview outputs and bindings remain review inputs before write authorization. These operations are currently present in source; final local gates for the integrated worktree remain pending confirmation at a fixed SHA. No provider run or human product acceptance is implied.

## Structured Brownfield sessions

Brownfield sessions persist selected discovery, per-scope status, Manager proposals/integration, explicit resolutions and model-only adoption records. Manual writes compare against the current session digest; Manager-run writes confirm a preview digest that also binds selected context, revisions, runtime fingerprint, and budget. Source and target revisions remain fixed. Stages accept closed JSON inputs and do not perform natural-language interviews or authenticate owner approval.

```text
markitect project brownfield --repo TARGET --source-repo SOURCE --brownfield-action start --revision TARGET_COMMIT --input START.json [--expect DIGEST --write]
markitect project brownfield --repo TARGET --source-repo SOURCE --brownfield-action begin --session ID --input BEGIN.json --expect SESSION_DIGEST --write
markitect project brownfield --repo TARGET --source-repo SOURCE --brownfield-action context --session ID --input CONTEXT.json
markitect project brownfield --repo TARGET --source-repo SOURCE --brownfield-action run --session ID --input RUN.json
markitect project brownfield --repo TARGET --source-repo SOURCE --brownfield-action run --session ID --input RUN.json --expect PREVIEW_DIGEST --write
markitect project brownfield --repo TARGET --source-repo SOURCE --brownfield-action propose --session ID --input PROPOSAL.json --expect SESSION_DIGEST --write
markitect project brownfield --repo TARGET --source-repo SOURCE --brownfield-action integrate --session ID --input INTEGRATION.json --expect SESSION_DIGEST --write
markitect project brownfield --repo TARGET --source-repo SOURCE --brownfield-action resolve --session ID --input RESOLUTION.json --expect SESSION_DIGEST --write
markitect project brownfield --repo TARGET --source-repo SOURCE --brownfield-action plan --session ID --input PLAN.json
markitect project brownfield --repo TARGET --source-repo SOURCE --brownfield-action apply-adoption --session ID --input APPLY.json --expect SESSION_DIGEST --write
markitect project brownfield --repo TARGET --source-repo SOURCE --brownfield-action resume --session ID
```

The staged loop binds Manager work explicitly. `begin` takes a bare `ReverseIterationRequest`; the input contains manager ID, evidence IDs, purpose, review, and optional parent/superseded iteration IDs. The read-only `context` action takes `{"iterationId":"ITERATION_ID"}` for proposal context or `{"iterationId":"ITERATION_ID","phase":"integrate"}` for bounded parent context. Proposal context contains selected source evidence, the actual responsible Manager (including a parent-proposed Manager when present), and accepted or proposed public neighbor contracts. Integration context adds completed direct-child reports, public contracts, and bound digests. Neither context includes the full session or readiness report.

The normal Manager runner is `--brownfield-action run`. Its closed JSON input is `{"iterationId":"ITERATION_ID","phase":"propose","agentManagerId":"ACCEPTED_RUNTIME_MANAGER_ID"}`; use phase `integrate` for parent integration. The runtime mapping names an accepted Manager or configured accepted ancestor allowed to execute the proposed responsibility; the iteration's Manager remains the actual work identity. Each Manager phase is a separate invocation with bounded, phase-specific context. Without `--write`, `run` is a no-call preview and returns `previewDigest` plus the current `sessionDigest`. For one execution, pass that exact `previewDigest` via `--expect` and add `--write`. The digest binds the session, stage, selected context, source/target, runtime fingerprint, and budget; the service rechecks these under the session execution lock before recording the attempt and calling the provider. Results expose only the validated Manager proposal or integration, durable attempt status/receipt, and resulting session digest, never the whole session or raw logs.

The parent proposes direct child responsibilities explicitly. Begin a separate iteration for every adopted child, with its parent iteration ID, then repeat context and `run` for its proposal; continue recursively. A parent can integrate only when every direct child has proposed and every non-leaf child has integrated. The integration context contains the completed reports/contracts/digests needed to bind that work. There are no automatic retries: `retryOfAttemptId` is optional and valid only for the latest known failed attempt with a final receipt; it remains subject to configured `MaxRetries` and cumulative starts, duration, and cost. An uncertain attempt is not replayable. Manual `propose`/`integrate` remain provider-free compatibility stages; `iterate` remains a convenience action.

`resolve` input contains the iteration ID and a `Resolution`; an authorized owner or a Manager within authority delegated by the Work Item may record the decision and set a scope to `modeled` or `transitional`. Ask the contributor only when the decision is outside delegated authority, materially unresolved, or repository policy requires human review. The recorded actor/authority/provenance is a caller assertion, not human identity authentication. Only actual guarded adoption sets a scope to `adopted`.

`plan` is a read-only preview. To apply, `APPLY.json` contains `{"iterationId":"ITERATION_ID","expectedPlanDigest":"PLAN_DIGEST"}` copied from that preview. The caller also supplies `--expect` with the prior session digest and `--write`. Host recomputes the exact target-bound plan and current schema/build bindings, applies only the planned model-file edits through the guarded writer, and records the internally produced receipt under session compare-and-swap. `record-adoption` is explicitly rejected because a caller-supplied receipt is not trusted. If the model changes, session readiness remains false until the adopted model is committed and accepted under policy, both fixed source and target bases are current, and current full-repository coverage is conforming. Readiness considers only the active, unsuperseded iteration tree; old ledger history is retained but does not gate the new active tree. If source Apply succeeds but recording its session receipt fails, inspect both target model and session ledger before any retry; the CLI deliberately refuses to rerun automatically. A full Verify/Apply acceptance path after Brownfield model adoption remains distinct and requires current evidence.

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

The bundle identifies changed declarations, impact, provenance supplied by the caller, and Manager-specific briefings with public neighbor contracts. It compares the model; it does not infer meaning from prose. The provenance reference is a caller declaration, not authenticated identity. Dismissing an event changes its visibility only; it does not resolve a question, remove a Manager obligation, or convert failed evidence to a pass. Full-coverage model-edit plans reject draft model changes; accept and commit the model separately. Plan-bound scoped briefs are included in Manager, Reviewer, and full-Verify contexts, and fresh bindings plus Apply reject changed briefing history. An existing briefing store with a new model digest blocks work until the new change is briefed. The source now has `EnsureAcceptedHistory`, which bootstraps a committed baseline and scans complete first-parent history to append committed semantic model transitions and reversions; guided Explore, Plan and Full Verify invoke it. The initial model is a baseline and code-only commits do not fabricate model changes. Successful `project deliver` records immutable resolution evidence only after full Manager Verify and guarded Apply on the bound candidate; this does not authenticate a human or turn dismissal into resolution. The operation requires complete, unambiguous Git ancestry, and its integrated local gates remain pending current fixed-SHA confirmation.

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

Current source includes deterministic subprocess fixtures for Run/Verify/guarded Apply, durable Explore and Readiness, structured Brownfield sessions/context, and composed delivery. These fixtures establish protocol behavior only. The final integrated local gates remain pending confirmation at an immutable source SHA. Real Codex/Claude provider runs remain NOT RUN here, and neither fixtures nor source availability establish semantic correctness, complete product acceptance, human approval, or productivity gains. Historical validation reports retain their tested SHAs and outcomes; they are not rewritten as evidence for this worktree. See the [native work-item delivery checklist](design/project-world/native-work-item-delivery.md) for remaining user-journey gates.
