# Markitect agent entrypoint

**Current owner scope, 23 October 2026:** the [latest V1 clarification](docs/concepts/sources/v1-scope-clarification-20261023.md) supersedes conflicting v0.16 and historical directions below. Preserve model/compiler/registry/ownership/accounting semantics; the runtime may be replaced. Every Assessment, Implement and Verify uses the complete model-derived Manager hierarchy in fresh Run environments. Extract, Cockpit/UI, interactive authoring and old-consumer migration are excluded; the [current plan](docs/implementation-plan.md) specifies their bounded retirement. Read the [counter-review](docs/development/v1-architecture-audit-20261023.md) and [proposed architecture](docs/design/v1-execution-architecture.md). The [owner has approved coordinated source implementation](docs/concepts/sources/v1-source-implementation-approval-20261023.md), bounded provider/container-free tests, independent reviews, PRs and integration. Use at most two writing chats, Sol High, with exclusive worktrees/scopes. The subsequent standing evidence authorization lets the coordinator dispatch useful, concretely bounded real proofs without another owner confirmation; record source, limits, abort, cleanup and actual results. No release/deployment. Verify the [source archive](docs/development/v1-source-archive-20261023.md) before retirement.

Markitect is model-first development with delegated realization. People maintain a canonical model; Managers realize their own areas and review and integrate their children. The accepted target is [Agent workspaces and reviewed handoffs](docs/design/canonical-model/manager-apply-workflow.md), [DEC-067](docs/concepts/register.md#dec-067-agent-workspaces-and-reviewed-handoffs): one container and own working copy per Agent assignment, no domain Turns, direct delegation/questions and upward Git handoffs. Responsibilities are working rules; per-write protection and a custom evidence protocol are not prerequisites. Before assessing product direction, read [Markitect in brief](docs/vision.md#markitect-in-brief), [Common misreadings](docs/vision.md#common-misreadings) and the [concept register](docs/concepts/register.md). Do not reopen an accepted entry without new evidence.

**Source boundary, 23 October 2026:** Main `447ac3b152de79b52de80513e14dc94349a060bd` wires public Apply to assignments, with separate focused reviews and an Apply-wide Git lifetime. Legacy Pass/Turn APIs and excluded consumers remain in source until the approved retirement cut. Canonical self-model and executable guides have not yet migrated. Historical work orders do not authorize conflicting implementation or further provider runs.

## Every task

**Container lifetime:** the current target is one fresh Manager environment per Manager per Run and one Git Workspace Service per Run. The existing runtime guide is source history, not a new work order. Candidates and mandatory results must survive complete resource closure. No withdrawn reuse experiment is restarted.

1. Read the [working method](docs/development/working-method.md), current [delivery plan](docs/implementation-plan.md) and bounded assignment. Use an isolated worktree from its integrated base and a non-protected branch with a scope prefix such as `dev/`, `fix/` or `docs/`. Verify path, branch, full SHA and local changes before writing. Preserve other sessions' work.
2. Read [.markitect/model.yaml](.markitect/model.yaml), then run actual structural preparation and select the Package that owns the change:

   ```text
   go run ./cmd/markitect check --repo .
   go run ./cmd/markitect context product --repo .
   ```

   The canonical v1 model holds intent; exact artifact paths and Definition links live in `.markitect/registry/`. Context reports selected Definitions, provenance and inclusion reasons. Structural preparation does not start sessions, execute project checks or establish runtime readiness. **Source-only historical execution description:** the following Apply/Extract details describe existing source and do not dispatch the new core; current scope and authority come only from the [current plan](docs/implementation-plan.md). Apply ([DEC-056](docs/concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace) with DEC-058 to DEC-060) and a first Extract are implemented ([runtime contract](docs/design/canonical-model/manager-apply-runtime.md)), pending owner acceptance. Read the [workflow](docs/project-workflow.md) for explicit manifests and dependencies.

   Run Apply only with explicit authority: `go run ./cmd/markitect apply --repo . --runtime-profile ABSOLUTE_RUNTIME_YAML --run-config ABSOLUTE_RUN_JSON` over a committed repository; it starts real Agents and needs a container engine and credentials. Its JSON report names the evidence directory, `<state>/runs/<runID>/` (`events.jsonl`, `transcripts/`, `report.json`). A Run is closed: no question or answer crosses Root to the invoking AI or the user; a model that lacks a material decision or contradicts itself ends the run Failed without publication ([DEC-063](docs/concepts/register.md#dec-063-the-model-is-authoritative-defects-escalate)). Apply publishes only a verified candidate and never commits. The [workflow](docs/project-workflow.md#execute-verify-and-apply) has the sequence.
3. Implementation-only work needs no intent change. A new artifact needs one exact registration in its owner's registry shard, with existing Definition links and a reason. An exceptional ignored file needs an exact observed path and reason in `.markitect/ignore.yaml`; Gitignore does not supply accounting. Canonical and control inputs cannot be ignored or registered as artifacts.
4. After changing `.markitect/model/`, preview the official readable view and write it using the returned `view.previewDigest`:

   ```text
   go run ./cmd/markitect docs --repo .
   go run ./cmd/markitect docs --repo . --expect PREVIEW_DIGEST --write
   ```

   Do not edit generated views. The model digest, rendered `contentDigest` and source/target-bound `previewDigest` have different meanings. Changed sources or targets require a fresh preview.
5. Before a PR, run the relevant local checks in [CONTRIBUTING](CONTRIBUTING.md#verify-a-change) and obtain independent complete-diff review. Freeze the reviewed head and follow the [development integration policy](docs/implementation-plan.md#integration): automatic CI is paused, full CI is available manually, and merges do not wait for hosted CI by default. Actual Main composition and Main read-back remain distinct requirements. Source tests and smokes are bounded evidence, not provider, Apply, release or owner acceptance.

## Where things are

- [Delivery plan](docs/implementation-plan.md): scopes, dependencies, reservations and integration rules; chats and PRs hold progress and evidence.
- [Code map](docs/development/code-map.md): every actual Go package and its owning guide.
- [Generated model view](docs/markitect/project.md): readable canonical Definitions.
- [Concept register](docs/concepts/register.md): accepted decisions and open questions.
- [Documentation map](docs/README.md): other documents.
