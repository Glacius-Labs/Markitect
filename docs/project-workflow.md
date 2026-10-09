# Project workflow

`markitect project` is the project-owned entry point for explicit managers, statements, expected artifacts, checks, and bounded implementation work. It complements the existing Project/Domain commands; the historical top-level `init` and the `prepare` / `copy-me` evidence-capture flow remain available with their existing meanings.

An AI author can inspect the active project-model contract with `markitect project schema`; this read-only command needs no repository or Markitect source checkout and emits the exact schema used by project validation.

The [9 October user direction](design/project-world/model-first-user-workflow.md) specifies the next workflow: contributor-provider onboarding, guided Explore until the first successful Apply, configurable Managers, generated documentation under a configurable root `docs/`, complete file classification and iterative reverse adoption before cleanup. Those are requirements, not additional commands or behavior provided by the current CLI described below.

## Choose the executable

The published CLI and a source candidate are different tools. Install or upgrade the released CLI only through the documented versioned, verified release bundle. Until this project workflow is released, run it from the exact Markitect source checkout being evaluated:

```powershell
go run ./cmd/markitect project init --help
```

Building or running that candidate does not replace the global executable and does not create or change a project-local tool pin. A built candidate binary has the same property. Do not treat a locally built binary as a published release. In the command examples below, `markitect` means that candidate binary; when using `go run`, prefix each command with `go run ./cmd/markitect`. An existing supported agent host and its account/runtime are prerequisites for conversational work; the Markitect CLI is deterministic software, contains no language model, and does not change provider, editor, hook, plugin, MCP, or account configuration.

## Start or adopt a project

For a new repository, preview the files Markitect will own, then initialize on a writable feature branch:

```powershell
markitect project init --repo . --name shop-cancellation
markitect project init --repo . --name shop-cancellation --write
markitect project check --repo .
markitect project index --repo .
markitect project document --repo .
```

The initialization owns only `.markitect/`: the project manifest, empty runtime placeholder, root manager, generated readable view, and local ignore rules. It does not configure an agent. The view is generated from the selected model and is safe to read or edit as a proposal, but the selected YAML model remains the compiled source of meaning.

Set up a local runner without writing YAML by hand. `setup` discovers a direct native provider executable, Python, and the adapter under the explicitly named Markitect source root; it invokes only `--version`, pins their paths and digests, and builds one mapping per active Manager. Codex discovery rejects npm `.cmd`/PowerShell wrappers and looks for its vendored native executable; use `--provider-executable` if discovery cannot find it. The provider account and supported host/CLI must already exist. Markitect never reads credentials or probes login status, so the report says authentication is `not-verified`.

Preview the ordinary runtime edit first. Supply the model and pricing weights chosen by the caller's budget policy; these estimate accepted usage and do not represent a provider quote, invoice, or hard billing cap:

```powershell
markitect project doctor --repo . --tool-root C:\src\Markitect --provider codex
markitect project setup --repo . --tool-root C:\src\Markitect --provider codex --model MODEL --effort high --input-micros-per-million INPUT_RATE --output-micros-per-million OUTPUT_RATE --max-cost-micros TASK_BUDGET
```

Review `discovery`, `mutation`, and `editPlan` in the JSON preview, including every pinned executable and its version/digest. Then pass the exact `editPlan.digest` back to the same command:

```powershell
markitect project setup --repo . --tool-root C:\src\Markitect --provider codex --model MODEL --effort high --input-micros-per-million INPUT_RATE --output-micros-per-million OUTPUT_RATE --max-cost-micros TASK_BUDGET --expect EDIT_PLAN_DIGEST --write
markitect project check --repo .
git add .markitect/runtime.yaml
git commit -m "Configure project-local Markitect runtime"
```

`setup` runs no agent and changes no global provider, editor, hook, plugin, MCP, or account configuration. `doctor` checks tool/version availability, declared-check executable names on `PATH`, and whether the repository is on a named feature branch; it does not establish account authentication. The runtime uses `controlled-local`, a finite depth/start/time/output/candidate budget and a caller-supplied cost estimate. It allows one bounded retry for known invalid proposals or model-compiler diagnostics; uncertain provider/transport outcomes, blocked work, and stale state are not retried. Retry counts and accepted token usage appear in the run receipt; cost weights remain estimates, not an invoice cap. It does not claim OS isolation. Once selected model and runtime inputs are accepted, commit them before `project plan`; planning compares fixed `HEAD` to the optional `--since` baseline while executing against the current committed project.

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

The agent host is the conversation surface; Markitect is the source of project context and the boundary for structured proposals. A practical loop is:

1. Ask for a change in ordinary language, such as “allow cancellation only before shipment and release the reservation in the same transaction.”
2. Have the agent read the relevant manager context and current index/check report. Use `project impact --base BASE_COMMIT --revision CANDIDATE_COMMIT` when comparing fixed revisions.
3. If the business model or source inventory changes, ask the agent to return a closed mutation JSON proposal for `project edit --input ...`. A user or root Manager may adjust the manifest's inventory roots or exclusions; narrower managers cannot broaden project inventory. Use `project setup` to preview and bind the standard local provider runtime, or let the user/root Manager propose a custom runtime mutation through `project edit`. Markitect renders the exact edit plan and digest; within the original user-authorized task, the agent can apply that same proposal with `--expect EDIT_PLAN_DIGEST --write`. There is no need to hand-edit YAML to propose a model, scope, or standard Codex/Claude setup.
4. After model, scope, adoption, or runtime changes, review the candidate and commit the accepted changes locally on the feature branch. Markitect does not commit. The selected checkout must reproduce the committed bytes and modes; keep line endings and Git checkout filters consistent with that requirement. Execution plans bind a fixed Git HEAD and the currently selected project inputs, so planning rejects uncommitted or otherwise different inventoried bytes instead of silently verifying a different source state.
5. Review manager assignments and artifacts, then plan implementation. `--since OLD_COMMIT` compares impact from a fixed earlier project basis to the current committed project; the plan's execution basis remains current `HEAD`. It helps tell the agent what changed without making the old commit the code being edited. Without `--since`, no change-baseline comparison is requested. Planning is read-only unless `--write` records the exact, already-authorized request. Persisting a plan does not start an agent.
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

Conversational integration requires a supported host/runtime to make the structured Markitect calls. The protocol example under `examples/project-world/.markitect/agent-fixtures/` is only a fixture; it is not proof of a provider call, verified agent setup, operating-system isolation, or agent compliance. Repository instructions can make the workflow discoverable but cannot technically prevent an agent with ordinary write access from bypassing it. Use the reported execution mode and its evidence, and keep human acceptance separate from structural checks.

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

## Executable example

`examples/project-world/` models a Shop with a root manager, commerce integration manager, Sales manager, and separate Orders and Inventory managers. Its vertical slices cover order state, reservation release, the shared transaction rule, declared artifacts, and a literal test command. The Python example uses only the standard library and SQLite:

```powershell
cd examples/project-world
python -B -m unittest discover -s tests -v
```

The cancellation operation rejects shipped orders, is idempotent for an already cancelled order, releases the reservation in the same transaction, and rolls back when release fails. The tests establish those finite cases only; they do not prove production database behavior, complete requirements, agent compliance, or a productivity gain. The empty runtime file intentionally leaves agent/provider setup to the project owner.
