# Project workflow

`markitect project` is the primary model-first project workflow: owners maintain one recursively managed project model under `.markitect/`, Managers own conceptual slices and their artifact responsibilities, and agents implement bounded work against that model. The structural compiler, schema and rendering tools also serve Markitect's own engineering resources. Versioned release records describe their actual historical behavior; compatibility alone is not a retention requirement.

An AI author can inspect the active project-model contract with `markitect project schema`; this read-only command needs no repository or Markitect source checkout and emits the exact schema used by project validation.

The installed native entrypoints identify `.markitect/project.yaml` as the selector for the active YAML model and policy. Inspect `markitect project index --repo PATH` for its current Manager identities and `markitect project check --repo PATH` for structure and coverage before preparing a guarded `project edit`. The empty initialized root is a starting point: the responsible Manager defines the task's concepts, artifact ownership and checks within delegated authority. New projects use `markitect project init`. The obsolete top-level initializer has been removed.

The current source provides guided project initialization, native onboarding, durable one-scope Explore records, scoped readiness evaluation and digest-bound structure acknowledgements, model editing, configurable readable documentation, full repository classification, Manager execution, integration, verification, guarded Apply and recovery. New projects use `workflowMode: guided` and `acceptancePolicy: committed-model`; legacy manifests with empty fields remain readable. Guided plans bind an exploration and named scope. Explore/readiness do not start an agent or authenticate the person whose decision reference is recorded. The mandated ordinary-contributor journey and its completion gates are captured in the [native work-item delivery checklist](design/project-world/native-work-item-delivery.md); that design checklist is a plan, so its gates remain distinct from what source and runtime evidence currently establish.

For scoped, read-only graph queries over the selected model and explicitly selected records, see [Project knowledge](project-knowledge.md).

```text
project explore --repo PATH [--exploration ID | --input .markitect/drafts/exploration.json] [--expect PLAN_DIGEST --write]
project readiness --repo PATH --exploration ID --scope ID [--acknowledge-structure --actor ACTOR --authority TEXT --decision-ref REF --acknowledged-at RFC3339 --expect DIGEST --write]
project plan --repo PATH ... --exploration ID --scope ID
```

The current source exposes the composed delivery operation as `project deliver --repo PATH --exploration ID --scope ID [--run ID] --write`. It requires explicit write authorization, plans only a ready scope, and advances the durable run through Manager execution, integration, Verify, preflight and guarded Apply. If interrupted, pass the returned run ID to resume that proof; a durable Apply receipt can recover scope completion. Review the preview and all bindings before authorizing writes. This is a bounded source workflow, not proof of semantic correctness, human approval, or a published-release capability. See the exact current options in `markitect project --help` from the matching source checkout.

Brownfield work also has a structured session API for iterative model proposals and selected adoption. Each stage consumes explicit JSON, binds source and target revisions, and writes only after preview and digest confirmation; this is a protocol surface, not a natural-language interview or an authenticated owner decision:

```text
project brownfield --repo TARGET [--source-repo SOURCE] --brownfield-action start|begin|context|run|propose|integrate|iterate|resolve|plan|apply-adoption|resume [--revision COMMIT] [--session ID] [--input .markitect/drafts/brownfield-stage.json] [--expect DIGEST --write]
```

JSON transport records passed as `--input`, `--request` or `--output` must use normalized repository-relative slash paths ending in `.json` under `.markitect/drafts/` or `.markitect/runs/`. Absolute paths, backslashes and root-level JSON names are rejected. Explore, project edit and Discovery records are rooted in `--repo`. Brownfield stage inputs and its durable session are rooted in `--source-repo` when supplied, otherwise `--repo`; `--repo` remains the target project/runtime root. The agent creates the required records in that selected repository from current command outputs and decisions.

The staged session actions separate assignment (`begin`), Manager proposal, parent integration, owner resolution, adoption planning and guarded application. `propose` and `integrate` accept explicit provider-free reports for independently supplied evidence; they do not run an agent. For the normal agent workflow, `run` defaults to a no-call preview and accepts closed JSON such as `{"iterationId":"ITERATION_ID","phase":"propose","agentManagerId":"ACCEPTED_RUNTIME_MANAGER_ID"}`. The runtime mapping identifies an accepted Manager or configured accepted ancestor that can execute the proposed responsibility; the iteration Manager remains the actual work identity. Each Manager phase is a distinct bounded invocation with its own context and durable attempt. To execute exactly that preview, pass its returned `previewDigest` as `--expect` and add `--write`; a changed session, source, target, Manager assignment, runtime configuration, or budget makes the preview stale. The result returns the Manager artifact and attempt receipt/status, not the complete session. There is no automatic retry: `retryOfAttemptId` may name only the latest known failed, receipted attempt, subject to configured retry and cumulative time/start/cost bounds; an uncertain attempt cannot be replayed.

Begin with a root iteration. Read its `context`, run its `propose` phase, and let its proposal assign direct child Managers and evidence explicitly. For each adopted child, create a separate child iteration with its parent iteration ID and repeat context and `run` with phase `propose`; recurse through the proposed tree. A parent may integrate only after every direct child has proposed and each non-leaf child has integrated. Retrieve the bounded parent context with `context` input `{"iterationId":"PARENT_ITERATION_ID","phase":"integrate"}`; it includes completed direct-child reports, contracts, and bound digests. Run the parent's `integrate` phase only after reviewing that context. Do not hand-compose Manager proposals or integrations in the ordinary workflow when the Manager runner is available.

After integration, an authorized owner or a Manager acting within authority delegated by the Work Item records the explicit resolve decision and may mark a scope modeled or transitional. Ask the contributor only when the decision is outside delegated authority, materially unresolved, or repository policy requires human review. Caller-supplied actor, authority, and decision reference record provenance but do not authenticate a human. Plan is read-only. After reviewing that plan, apply-adoption requires both its plan digest in a closed JSON input ({"iterationId":"…","expectedPlanDigest":"…"}) and the prior session digest via --expect; it recomputes the target-bound plan and build/schema bindings, performs guarded model-only Apply, then records its internally generated receipt with session compare-and-swap. Caller-supplied receipts are rejected; the old record-adoption action returns an error. Context is read-only, scoped to the requested Manager, and does not return the complete session or readiness report. Iterate remains a convenience action combining begin, proposal and optional integration.

Owner resolution can set a scope to modeled or transitional (for a deferred scope); it does not set the scope to adopted. Only successful `apply-adoption` records the scope as adopted. Readiness evaluates the active, unsuperseded iteration tree, so retained earlier iterations do not continue to block current readiness. A session cannot become ready until adoption has actually been applied, the resulting canonical model is committed and accepted under repository policy, both fixed source and target bases are current, and current full-repository coverage is conforming. These are current source contracts; final local package gates remain pending confirmation at a fixed SHA. The structured protocol does not establish natural-language interview quality or adopter acceptance. The [operations guide](project-operations.md) describes these operations and their limits.

The subsequent [operation scope and model briefing direction](design/project-world/operation-scopes-and-model-briefings.md) requires impact-scoped implementation but full Manager verification, model-preserving Cleanup, full-scope Reconcile, configurable review depth and persistent change briefings. The current source implements these operations for full-coverage projects, including standalone fixed-revision Verify and scoped briefings with visibility-only dismissal. Guided Explore, Plan and Full Verify reconcile accepted first-parent model history through `EnsureAcceptedHistory`; the first valid model is a baseline and subsequent committed model changes/reversions receive records. After successful full Verify and guarded Apply, `deliver` can record immutable, candidate-bound resolution evidence for applicable accepted-model events. Dismissal remains visibility-only; separate notification channels and human approval are not provided. The [work-item delivery checklist](design/project-world/native-work-item-delivery.md) defines further user-journey and validation gates; it is not itself evidence that those gates have passed.

### Brownfield evidence and Manager delegation

For each begin request, `evidenceIds` are the only raw source files supplied to that Manager and support its claim citations. `delegationEvidenceIds` is a required, disjoint metadata-only pool from which it may assign children. `[]` authorizes no child evidence; missing pools are rejected, including ambiguous stored sessions. Selected Discovery inventory is metadata (IDs, paths, bases and digests), not behavioral evidence or delegation authority. A child copies both exact arrays assigned by its parent proposal within the parent's declared pools. Non-leaf integration receives direct-child final reports, citations, public contracts and digests; descendant citations travel in a child's integration report without exposing raw child source.

Minimal root BEGIN.json shape; begin consumes the request directly:

    {"id":"root-review","managerId":"ROOT_MANAGER_ID","evidenceIds":["ROOT_OWNED_EVIDENCE_ID"],"delegationEvidenceIds":["CHILD_EVIDENCE_ID"],"purpose":"Model the observed behavior.","review":"Report evidence, uncertainty, and public contracts."}

Stage outputs are bounded for review. Session fields expose a safe overview of fixed bases, scope status, iteration identities, and digests, not raw sources or private report bodies. Resume and plan also intentionally retain typed readiness questions and conflict diagnostics so the outer coordinator can route or decide blockers; these contain no source/report bodies. Context returns only the requested Manager scope, and the inner Manager request contains only that Manager's bounded context (own selected content and authorized metadata) plus completed direct-child reports for integration, never the full session or readiness. Run returns the scoped proposal/integration and attempt receipt. Plan returns the full model AdoptionPlan for review before Apply. The per-session process lock is released by the operating system when its process exits. An ambiguous ledger publication or invocation with unknown cost/terminal outcome fails closed; inspect the overview and run journal rather than replaying or deleting the lock file. requestContractDigest binds stable schema and instructions while excluding cumulative remaining-budget counters that may advance after priced work; previewDigest still binds current ledger and budget and must be refreshed before a retry.

## Choose the executable

The current project workflow is in source; it is not included in the published v0.14.1 CLI. The documented versioned, verified bundle is evidence of that historical release; it is not an installation of current project commands. To use current `project` commands, build a candidate binary from the exact Markitect source checkout being evaluated or use `go -C MARKITECT_SOURCE_ROOT run ./cmd/markitect`.

```powershell
go -C C:\src\Markitect run ./cmd/markitect project init --repo C:\src\my-project --name my-project
```

The remaining commands assume the current directory is the target project root and `markitect` resolves to that candidate binary; use `--repo .`. When invoking source directly with `go -C C:\src\Markitect run ./cmd/markitect`, replace each `--repo .` with the exact target path because `-C` changes the command's working directory. Running a source candidate does not replace the global executable or change a project-local tool pin, and no new release is promised here. The project model in a committed revision is the accepted specification Markitect uses for that revision; drafts and uncommitted model changes are proposals. Git commit IDs, digests, provenance references, agent reports and passing checks do not authenticate human approval. The owner remains responsible for decisions and the project’s approval process.

## Start or adopt a project

For a new repository, preview the model-first files Markitect will own, then initialize on a writable feature branch:

```powershell
markitect project init --repo . --name shop-cancellation
markitect project init --repo . --name shop-cancellation --write
markitect project check --repo .
markitect project index --repo .
markitect project document --repo .
```

Initialization creates the project manifest, runtime placeholder, root Manager and local ignore rules under `.markitect/`, plus the readable view at `docs/markitect/project.md`. New manifests use `workflowMode: guided`, `acceptancePolicy: committed-model` and full repository coverage. YAML owns meaning; generated Markdown is the readable view.

Install the selected provider's native guidance before configuring execution. Preview `project onboard`, inspect its exact writes and conflicts, then apply the same digest:

```powershell
markitect project onboard --repo . --provider codex
markitect project onboard --repo . --provider codex --expect ONBOARDING_DIGEST --write
```

Codex receives `AGENTS.md` and ten skills under `.agents/skills/markitect-{init,extract,design,implement,cleanup,verify,apply,check,suggest,configure}/`; Claude receives `CLAUDE.md` and the same operation names under `.claude/skills/`. Skills have discoverable provider metadata and local supporting references. Entrypoints route directly to operations. Implement chains the needed design/check/verify/apply work within the task's authority; suggestions alone remain proposals. Existing user prose and unrelated skills are preserved. Onboarding can remove a wholly known obsolete Markitect router; customized same-name files cause an explicit conflict.

### Native Codex work

`project setup` configures the supported native Codex Manager workflow. It requires the existing account, actual Codex CLI 0.162.0, `gpt-6-luna`, `high` effort, the `luna-high` profile and installed Codex project guidance. Claude Manager execution is not currently supported by setup; Claude instruction installation and tool diagnostics remain separate useful operations. No execution-mode selector or proposal-only Manager fallback remains.

Setup discovers the direct native executable, Python and adapter under the explicitly named Markitect source root; it runs only version discovery and pins paths, modes and digests. Npm shell wrappers are rejected; discovery resolves the vendored executable, or the caller supplies `--provider-executable`. Credentials are not read and authentication remains `not-verified` until actual provider work. Supply explicit input/output budget weights and a maximum cost estimate; these are local accounting weights, not a price quote, invoice or hard billing cap.

```powershell
markitect project doctor --repo . --tool-root C:\src\Markitect --provider codex
markitect project setup --repo . --tool-root C:\src\Markitect --provider codex --model gpt-6-luna --effort high --codex-profile luna-high --input-micros-per-million INPUT_RATE --output-micros-per-million OUTPUT_RATE --max-cost-micros TASK_BUDGET
# Inspect discovery, mutation and editPlan, then use the exact editPlan.digest:
markitect project setup --repo . --tool-root C:\src\Markitect --provider codex --model gpt-6-luna --effort high --codex-profile luna-high --input-micros-per-million INPUT_RATE --output-micros-per-million OUTPUT_RATE --max-cost-micros TASK_BUDGET --expect EDIT_PLAN_DIGEST --write
markitect project check --repo .
```

Commit the initialized model, generated instructions and runtime before `project plan`. Runtime instruction pins must match both the fixed accepted revision and current bytes. If onboarding or guidance changes, regenerate the setup preview and pins, accept the guarded edit, then commit the coherent model/instructions/runtime together. Planning uses fixed `HEAD` and the optional `--since` baseline.

Each Manager receives bounded model/artifact inputs and exact pinned instructions in a fresh candidate file tree with ordinary file, shell and test tools. Model-declared Managers are scheduled once by Host. The inner worker is responsible for its candidate; the outer conversational agent owns canonical editing and repository Git operations. Candidate trees currently contain no Git metadata/history. Native tool access follows existing profile, account and rules with caller permissions; this is controlled local operation, not OS isolation. Setup changes no global provider, editor, hook, plugin, MCP or account configuration.

Candidate collection checks actual changes against ownership, allowed paths, immutable instructions, control-plane exclusions and fixed source. The adopting checkout is written only by guarded Apply after freshness checks, required checks and separate read-only reviews. Typed native receipts bind changed paths and a recomputed delta digest; full workspace digests remain adapter claims bound to stdout. Deletion and binary deltas are unsupported. Native helpers are disabled because current CLI JSONL lacks complete lifecycle accounting; a nonzero helper limit fails before launch. `helperStarts: 0` describes that configuration, not a census of operating-system processes.

Runtime limits bound depth, starts, time, output, candidates and estimated cost. One bounded retry can address known invalid proposals or actionable compiler diagnostics; uncertain provider/transport outcomes and stale state are not blindly retried. The [native-work contract](design/project-world/native-codex-work.md) and [dated validation](validation/project-native-work-2026-10-09.md) separate source checks, actual provider evidence and remaining limits. Reviews remain evidence, not human acceptance.

For an existing repository, adoption begins from a full fixed source commit and an explicit list of regular-file paths, reasons, exclusions, and scope roots in a JSON request. Discovery emits a bound record. P1 `distill --report` validates a supplied report and starts no provider. If you deliberately want one agent-assisted proposal, first configure and commit the runtime, then use `distill --generate --write`; it runs the root Manager's configured agent once on the selected frozen evidence, requires caller-supplied pricing/budget weights and provider-reported usage, and writes a proposal plus a separate receipt under `.markitect/drafts/`. The report is still unverified model input, not an acceptance. `adopt` shows the selected model proposal before a separate guarded write:

First initialize the target repository and commit that scaffold on its feature branch. If adoption should inventory the existing code, have the user or root Manager propose the desired inventory roots and exclusions through `project edit`, review/apply that exact edit, and commit it too. Adoption binds its model-only proposal to a committed target project basis; it does not infer an uncommitted or provisional model, nor silently broaden selected inventory. If the repository is already a Markitect project, use its existing committed project basis.

```powershell
markitect project discover --repo . --request .markitect/drafts/discovery-request.json --output .markitect/drafts/discovery.json
markitect project distill --discovery .markitect/drafts/discovery.json --report .markitect/drafts/distillation.json --repo . --output .markitect/drafts/validated-distillation.json
# Optional one-call generation, after setup and committing the runtime:
markitect project distill --repo . --discovery .markitect/drafts/discovery.json --generate --write --output .markitect/drafts/distillation.json --input-micros-per-million INPUT_RATE --output-micros-per-million OUTPUT_RATE --max-cost-micros TASK_BUDGET
markitect project resolve --repo TARGET_REPO --source-repo SOURCE_REPO --revision TARGET_COMMIT --discovery .markitect/drafts/discovery.json --report .markitect/drafts/distillation.json --input .markitect/drafts/owner-choices.json --output .markitect/drafts/resolution.json
markitect project adopt --repo . --source-repo . --revision TARGET_COMMIT --discovery .markitect/drafts/discovery.json --report .markitect/drafts/distillation.json --resolution .markitect/drafts/resolution.json --output .markitect/drafts/adoption-plan.json
# Review the exact plan JSON and its digest, then:
markitect project adopt --repo . --source-repo . --revision TARGET_COMMIT --discovery .markitect/drafts/discovery.json --report .markitect/drafts/distillation.json --resolution .markitect/drafts/resolution.json --plan .markitect/drafts/adoption-plan.json --expect REVIEWED_PLAN_DIGEST --write
```

The request, report, owner choices, and resolution are explicit transports, not implicit guesses from the working directory. `resolve` accepts only `actor`, `authorityClaim`, `decisionReference`, `questions`, and `scopes`; it derives the target basis, schema/build digests, proposal/discovery bindings, and `authenticated:false` from the active source and target. Every scope needs an explicit `adopt` or `defer` choice with a reason; each adopted scope's questions need answers. The resolution is an unauthenticated decision record, not proof of identity or automatic acceptance. Review its digest before using it in `adopt`. Generated distillation preserves a separate execution receipt and cost estimate; caller-supplied prices are only budget weights, never a guaranteed bill ceiling. Its claims and questions remain proposals. Deferred scopes contribute no model files; changing any bound source, proposal, resolution, or target basis makes the plan stale. These operations write Markitect-owned files only.

Example `owner-choices.json` (replace IDs with those in the validated report):

```json
{
  "actor": "user",
  "authorityClaim": "The project owner selected the cancellation scope for adoption review",
  "decisionReference": "shop-decision-17",
  "questions": [
    {
      "questionId": "cancellation-timing",
      "scopeId": "orders",
      "disposition": "answer",
      "answer": "Cancellation is allowed only before shipment",
      "reason": "The owner confirmed the current business rule"
    }
  ],
  "scopes": [
    {"scopeId": "orders", "status": "adopt", "reason": "The owner selected Orders"},
    {"scopeId": "inventory", "status": "defer", "reason": "Inventory requires a separate review"}
  ]
}
```

The `resolve` command writes only the resolution transport under the source repository's `.markitect/drafts/` (or prints it to stdout). It does not write target model/runtime files, invoke a provider, or accept the model proposal.

## Everyday conversational work

The agent host is the conversation surface; Markitect supplies project context and validates structured proposals. A practical current-source loop is:

1. Ask for a change in ordinary language, such as “allow cancellation only before shipment and release the reservation in the same transaction.”
2. Have the agent read the relevant manager context and current index/check report. Use `project impact --base BASE_COMMIT --revision CANDIDATE_COMMIT` when comparing fixed revisions.
3. If the business model or source inventory changes, ask the agent to return a closed mutation JSON proposal for `project edit --input ...`. A user or root Manager may adjust the manifest's inventory roots or exclusions; narrower managers cannot broaden project inventory. Use `project setup` to preview and bind the standard local provider runtime, or let the user/root Manager propose a custom runtime mutation through `project edit`. Markitect renders the exact edit plan and digest; within the original user-authorized task, the agent can apply that same proposal with `--expect EDIT_PLAN_DIGEST --write`. There is no need to hand-edit YAML to propose a model, scope, or standard Codex/Claude setup.
4. After model, scope, adoption, or runtime changes, review the candidate and commit the accepted changes locally on the feature branch. Markitect does not commit. The selected checkout must reproduce the committed bytes and modes; keep line endings and Git checkout filters consistent with that requirement. Execution plans bind a fixed Git HEAD and the currently selected project inputs, so planning rejects uncommitted or otherwise different inventoried bytes instead of silently verifying a different source state. The native agent must be able to complete required Git writes through its client's ordinary approval path. If that action is denied, preserve the exploration and draft and report the exact blocker; do not write implementation files directly to bypass accepted-model readiness. Resume the same exploration after the ordinary action completes.
5. Review manager assignments and artifacts, then plan implementation. `--since OLD_COMMIT` compares impact from a fixed earlier project basis to the current committed project; the plan's execution basis remains current `HEAD`. It helps tell the agent what changed without making the old commit the code being edited. Without `--since`, no change-baseline comparison is requested. Planning is read-only unless `--write` records the exact request; persisting a plan does not start an agent. In guided projects, the plan binds the selected Explore record and named scope; readiness is evaluated explicitly rather than inferred from free-text conversation.
6. Start or resume the accepted plan, inspect its status, run the declared checks, and apply only the verified candidate bound to the expected branch and source state. If a required declared check fails, inspect the failure and use `project repair --repo . --run RUN_ID --write` to request the bounded repair on the same run ledger; without `--write`, `repair` only reports current status. Repair is limited to a known failed-check candidate, keeps the original run/plan and receipts, and cannot apply that unchanged failed candidate. Run fresh `verify` on the new candidate before applying it. Provider/transport uncertainty, blocked work, and stale state are not retried automatically.

```powershell
markitect project context --repo . --manager '["project.markitect.example.org/v1alpha1","Manager","commerce.sales.orders","orders"]'
markitect project plan --repo . --goal "Cancel confirmed orders and release their reservation atomically" --manager '["project.markitect.example.org/v1alpha1","Manager","commerce.sales.orders","orders"]' --manager '["project.markitect.example.org/v1alpha1","Manager","commerce.sales.inventory","inventory"]'
markitect project plan --repo . --goal "Cancel confirmed orders and release their reservation atomically" --since ACCEPTED_MODEL_COMMIT
markitect project plan --repo . --goal "Cancel confirmed orders and release their reservation atomically" --manager '["project.markitect.example.org/v1alpha1","Manager","commerce.sales.orders","orders"]' --manager '["project.markitect.example.org/v1alpha1","Manager","commerce.sales.inventory","inventory"]' --write
markitect project run --repo . --plan PLAN_ID --write
markitect project status --repo . --run RUN_ID
markitect project verify --repo . --run RUN_ID --write
markitect project repair --repo . --run RUN_ID
markitect project repair --repo . --run RUN_ID --write
markitect project verify --repo . --run RUN_ID --write
markitect project apply --repo . --plan PLAN_ID --run RUN_ID --candidate CANDIDATE_ID
markitect project apply --repo . --plan PLAN_ID --run RUN_ID --candidate CANDIDATE_ID --branch TARGET_BRANCH --head TARGET_HEAD --worktree TARGET_TREE_DIGEST --expect VERIFICATION_DIGEST --write
```

The read-only apply preflight prints the exact verification digest, target branch, HEAD, and working-tree digest for the named candidate. Pass those same values to the following `--write`; if any binding changes, Apply rejects the write. The precise plan and candidate identifiers come from Markitect's JSON reports. Each manager receives its own scoped context; child work is integrated into a candidate before independent declared checks run. `resume` reconciles persisted work without repeating completed invocations. A plan never invokes an agent, and an agent response cannot grant authority. `--write` on run/resume/verify exposes the existing user-authorized operation; it is not a second approval prompt. `apply --write` is guarded against the exact plan, run, candidate, branch, head, tree, and fresh verification.

Conversational integration requires a supported host/runtime to make the structured Markitect calls. The protocol example under `examples/project-world/tests/agent-fixtures/` is only a fixture; it is not proof of a provider call, verified agent setup, operating-system isolation, or agent compliance. Repository instructions can make the workflow discoverable but cannot technically prevent an agent with ordinary write access from bypassing it. Use the reported execution mode and its evidence, and keep human acceptance separate from structural checks. A committed model is the accepted repository specification for its revision; no CLI record authenticates the human who chose to commit it.

## Local implementation and review

Standard `project setup` configures separate reviewer invocations for the responsible Managers. Each leaf implementation, and any implementation edit made by a parent, passes through a read-only review of the actual scoped candidate and accepted model. Concrete findings return to the implementer within the configured review-round budget. A passing result goes to the Manager for integration. Manager integration may return bounded `reworkRequests` to its active direct children; the affected work is implemented, reviewed and integrated again.

The review context includes the required artifact interfaces of active direct children and explicit artifact contracts for the supplied files. If another Manager owns such a contract, its public requirements remain available without exposing private descendant models or additional implementation files. File ownership and write authority remain explicit.

Local review bytes follow file ownership and the Manager's recorded integration edits. An artifact spanning several Managers supplies interface context; its implementation files receive local review under their respective owners. Managers integrate the resulting contracts and candidates.

The review mandate is the Manager's own task, current phase and assigned obligations. The overall run goal supplies orientation. Public dependencies let the reviewer assess its scope's use or provision of interfaces; missing implementation bytes from other Managers and pending integrated Host checks are not local defects. Missing evidence needed to assess the assigned scope still requires an incomplete or escalated result.

Artifact descriptions of intended behavior or static test coverage do not by themselves claim that checks ran or passed. Integration summaries identify current child results separately from the Manager's own edits and remaining uncertainty. A compact summary omitting implementation details alone does not demonstrate a defect; targeted rework needs a concrete mismatch or unmet contract supported by current supplied evidence.

A pure routing Manager with no admitted implementation files, declared artifacts or recorded edits has no local implementation to review. The Host records `not-required` without starting a reviewer or inventing a pass. Its delegation and integration reports still have to close, and its children's implementation reviews remain mandatory. Missing declared artifacts or deleted implementation files do not qualify for this exception.

The project-owned `runtime.review` block contains reviewer `agents`, `maxRounds` and `maxManagerRounds`. Standard defaults are three cumulative reviews per Manager and phase, and two Manager rework rounds for the whole run, within the existing total start/time/cost limits. Manager rework and check repair retain those counters. All invocations remain charged to the same run; a new round never replenishes its budget. Legacy runtime files without this optional block keep the earlier execution path and do not claim local review evidence.

Review evidence binds the candidate and actual reviewed scope. Unchanged independent sibling results can remain valid after another child's correction; changed reviewed inputs require a fresh review. Reviewers have no write authority. Managers receive compact findings/results rather than worker transcripts. Fresh declared checks run on the integrated result before guarded Apply; neither reviewer agreement nor a Manager's completion report replaces those checks or owner acceptance. See the [implementation contract](design/project-world/delivery-contract.md#implementer-reviewer-schleife-und-gezielte-nacharbeit) for the role, budget and freshness boundaries.

Manager task contexts use `projectrun-task/v1`. During integration, inherited questions and risks remain open until explicitly resolved from current evidence. Remaining questions and risks, including required work in a sibling branch that this Manager cannot assess, require a partial report and escalation to the exact supplied parent target. Pending Host checks remain expected until Verify. A complete report cannot silently omit unresolved items. A valid direct-child rework request alone may accompany a complete integration report with no remaining questions or risks; the Host performs that repair and fresh integration before final run closure.

When an authorized ancestor resolves a forwarded obligation, the Host carries that decision back to its origin along the recorded forwarding chain. Exact provenance distinguishes unrelated questions with the same text. Partial resolutions retain other questions, risks and escalation state; absent legacy provenance is never reconstructed by guessing.

Targeted rework rejects a target that still has unresolved questions or risks. Its parent must explicitly resolve those obligations first; restarting implementation cannot silently erase them.

Plans and execution evidence bind the Host executable and schema. A changed Host build requires a new plan. Retained historical escalation status does not establish current provenance; outstanding legacy questions or risks without typed provenance block resolution.

## Executable example

`examples/project-world/` models a Shop with a root manager, commerce integration manager, Sales manager, and separate Orders and Inventory managers. Its vertical slices cover order state, reservation release, the shared transaction rule, declared artifacts, and a literal test command. The Python example uses only the standard library and SQLite:

```powershell
cd examples/project-world
python -B -m unittest discover -s tests -v
```

The cancellation operation rejects shipped orders, is idempotent for an already cancelled order, releases the reservation in the same transaction, and rolls back when release fails. The tests establish those finite cases only; they do not prove production database behavior, complete requirements, agent compliance, or a productivity gain. The empty runtime file intentionally leaves agent/provider setup to the project owner.

Native candidate trees contain no Git metadata or repository history. Repository Git operations belong to the outer project agent; a scoped candidate Git/history view remains unimplemented.
