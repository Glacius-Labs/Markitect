# Project workflow

Markitect manages an adopting repository through one canonical project model under `.markitect/`. The outer coding agent uses typed Markitect MCP operations for project decisions and lifecycle. The Host schedules the model's Managers and reviewers; those inner roles currently run through Codex CLI 0.162.0 App Server with the configured `gpt-6-luna` model at `high` reasoning effort. The published v0.14.1 binary does not include this `markitect project` workflow; use a matching source build.

## Connect and inspect

Install project-local guidance with `markitect project onboard`, then register the selected client's MCP server as described in [Provider adapters](provider-adapters.md). The server starts with one explicit repository root. It does not accept a tool argument that changes that root. Use the typed tools and their published schemas; do not invent record fields, IDs, or digests.

For an ordinary issue, bug, feature, or backlog Work Item, inspect the repository and Git state, then use `project_index`, `project_check`, and `project_context` as needed. Clarify only material questions about intent, scope, authority, or acceptance. Record assumptions and unresolved decisions rather than silently promoting them to requirements.

## Model and readiness

Use `project_explore` to preview a typed, one-scope Work Item. Review the selected model/revision binding and its returned `plan.digest`; persist that plan with `write: true` and the exact `expectedDigest`. This is durable task context, not accepted model intent.

If canonical intent, ownership, artifacts, rules, or checks need to change, use `project_edit` to preview a bounded mutation. Review structural findings and impact, repair concrete diagnostics within authority, and write only with the returned digest. Run `project_check` again. Under `committed-model` policy, readiness uses the valid selected model committed under repository policy. A digest or commit binds input bytes but does not authenticate human review.

Use `project_readiness` for the named scope. If structure acknowledgement is required, inspect its exact `writePlan`, then submit the acknowledgement with its returned digest and caller-supplied actor/provenance. This records an assertion; it does not authenticate identity. Do not add an acknowledgement step when the scope does not require it.

## Execute, verify, and apply

When the original Work Item authorizes execution and readiness is clear, use `project_deliver` with the same exploration and scope and `executeAuthorized: true`. It advances one durable plan/run through Manager work and integration, full verification, preflight, and guarded Apply. Inspect the returned stages and bindings. For staged control, use `project_plan`, `project_run`, `project_status`, `project_resume` or `project_repair`, `project_verify`, `project_preflight`, and `project_apply` with the typed fields returned by preceding stages. Apply only the reviewed verified candidate. Apply does not commit, merge, publish, deploy, or establish semantic acceptance.

Managers work in separate owned candidate workspaces. The Host integrates candidate deltas against the selected accepted base while retaining the source working-tree baseline; only guarded Apply writes the adopting checkout. A workspace is harvested or removed only after the root turn and every child start the adapter has observed are terminal. Native child telemetry is partial: observed counts are a lower bound, and zero observed child starts does not prove none occurred. Local execution uses the caller's permissions; the workflow is not an operating-system sandbox or hard billing cap.

If a run is interrupted, inspect `project_status` and continue the same run. Repair a known failed attempt only when the durable receipt makes it retryable and the existing budgets allow it. If the outcome is unknown, do not replay the role or create a replacement run; preserve the workspace/journal and use the recovery operation or report the unresolved blocker. Changed model, source, runtime, or candidate bindings require a fresh preview.

## Existing repositories

For Brownfield adoption, select fixed source and target revisions, explicit inventory paths, exclusions, and scope roots. `project_brownfield` and `project_brownfield_run` provide typed discovery/session stages and bounded Manager proposals/integration. Manager evidence IDs authorize raw source access; delegation IDs authorize only assignment to children. Keep those pools explicit and disjoint, include counterexamples and uncertainty, and treat observed code as evidence about implementation rather than accepted intent.

Brownfield adoption writes only the planned canonical model through guarded Apply. Review and commit that model before a separate implementation Work Item. Do not infer intended behavior from filenames, metadata, or a Manager report alone. Use `project_readiness` and current coverage/check results before execution.

## Acceptance and status

A successful compiler check establishes structural conformity; declared checks and scoped Manager/reviewer reports establish only their stated evidence. A successful Apply records technical materialization, not semantic correctness, owner acceptance, or human approval. The adopting repository owns its approval, merge, and release policy.

The [backlog](work-items/backlog.yaml) owns work status. Actual28 passed the combined A01 native smoke on source `cdd30b0efc540f151404dabe86b022275dc40d83`, including Verify, guarded Apply and original-turn recovery; the [A01 validation record](validation/a01-native-smoke-20261010.md) states what it does and does not establish. That result is source-bound; later CI and Main integration are separate evidence. Acceptance grants are exercise authority, not universal runtime defaults. Historical validation remains bound to its recorded source SHAs.
