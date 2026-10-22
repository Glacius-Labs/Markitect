# Markitect agent entrypoint

Markitect is model-first development with delegated realization. People maintain a canonical model; Managers realize their own areas and review and integrate their children. The accepted target is [Agent workspaces and reviewed handoffs](docs/design/canonical-model/manager-apply-workflow.md), [DEC-067](docs/concepts/register.md#dec-067-agent-workspaces-and-reviewed-handoffs): one container and own working copy per Agent assignment, no domain Turns, direct delegation/questions and upward Git handoffs. Responsibilities are working rules; per-write protection and a custom evidence protocol are not prerequisites. Before assessing product direction, read [Markitect in brief](docs/vision.md#markitect-in-brief), [Common misreadings](docs/vision.md#common-misreadings) and the [concept register](docs/concepts/register.md). Do not reopen an accepted entry without new evidence.

**Target/source boundary, 22 October 2026:** the owner commissioned concept documentation and a [concrete rebuild plan](docs/implementation-plan.md#apply-rebuild-plan), not implementation. Current code, canonical self-model and executable guides retain the shared-tree Pass implementation until explicitly migrated. Preserve existing bounded assignments and evidence; the new target is no provider or implementation grant. Read the owning target and plan before future Apply work rather than implementing conflicting old handoffs.

## Every task

**Container lifetime:** DEC-067 replaces the per-Turn target with one container per Agent assignment. The [runtime guide](docs/design/canonical-model/manager-apply-runtime.md#per-turn-containers) still describes the implemented per-Turn lifecycle; this documentation task does not change it or restart the withdrawn reuse experiment.

1. Read the [working method](docs/development/working-method.md), current [delivery plan](docs/implementation-plan.md) and bounded assignment. Use an isolated worktree from its integrated base and a non-protected branch with a scope prefix such as `dev/`, `fix/` or `docs/`. Verify path, branch, full SHA and local changes before writing. Preserve other sessions' work.
2. Read [.markitect/model.yaml](.markitect/model.yaml), then run actual structural preparation and select the Package that owns the change:

   ```text
   go run ./cmd/markitect check --repo .
   go run ./cmd/markitect context product --repo .
   ```

   The canonical v1 model holds intent; exact artifact paths and Definition links live in `.markitect/registry/`. Context reports selected Definitions, provenance and inclusion reasons. Structural preparation does not start sessions, execute checks or establish runtime readiness. Apply ([DEC-056](docs/concepts/register.md#dec-056-recursive-reconciliation-with-a-shared-workspace) with DEC-058 to DEC-060) and a first Extract are implemented ([runtime contract](docs/design/canonical-model/manager-apply-runtime.md)), pending owner acceptance. Read the [workflow](docs/project-workflow.md) for explicit manifests and dependencies.

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
