# Markitect agent entrypoint

Markitect is model-first development with delegated realization. People maintain a canonical model; a Manager hierarchy derived from it realizes, reviews and integrates the artifacts, and the host keeps a few clear technical guarantees.

**Current target:** [V1 concept 1.0](docs/concepts/v1-concept.md) ([DEC-068](docs/concepts/register.md#dec-068-v1-concept-10)). V1 is Model → Artifacts: Assessment, Implement and Verify each run through the complete model-derived hierarchy in fresh Run environments; Apply publishes only a stored, verified candidate after every Run has closed. Extract, Cockpit/UI, interactive authoring and Setup are not part of V1. V1 acceptance covers Codex with normal account login. The [delivery plan](docs/implementation-plan.md) owns the current steps and the known deviations of the code from the concept.

Before assessing product direction, read [Markitect in brief](docs/vision.md#markitect-in-brief), [Common misreadings](docs/vision.md#common-misreadings) and the [concept register](docs/concepts/register.md). Do not reopen an accepted entry without new evidence.

## Working rules

- At most two writing chats at a time, each with its own worktree and exclusive file scope. One writer at a time for shared contracts, registry shards, the self-model and generated files. Codex development chats use Sol High, as [approved](docs/concepts/sources/v1-source-implementation-approval-20261023.md).
- Real container, provider or Apply runs need a concrete bounded dispatch under the [standing evidence authorization](docs/concepts/sources/v1-source-implementation-approval-20261023.md#subsequent-standing-evidence-authorization): source, limits, abort, cleanup and actual result. Failed attempts stay recorded. No release or deployment.
- Fix host/agent interaction failures in host state or the tool contract, not with more prompt text ([concept §3.7](docs/concepts/v1-concept.md#37-tasks-not-process-d13)).
- Write all documentation in English ([DEC-069](docs/concepts/register.md#dec-069-documentation-is-written-in-english)). Verbatim owner statements in `docs/concepts/sources/` keep their original wording (record rule; see the DEC-069 scope notes).

## Every task

1. Read the [working method](docs/development/working-method.md), the [delivery plan](docs/implementation-plan.md) and your bounded assignment. Use an isolated worktree from its integrated base and a non-protected branch with a scope prefix such as `dev/`, `fix/` or `docs/`. Verify path, branch, full SHA and local changes before writing. Preserve other sessions' work.
2. Read [.markitect/model.yaml](.markitect/model.yaml), then run actual structural preparation and select the Package that owns the change:

   ```text
   go run ./cmd/markitect check --repo .
   go run ./cmd/markitect context product --repo .
   ```

   The canonical v1 model holds intent; exact artifact paths and Definition links live in `.markitect/registry/`. Context reports selected Definitions, provenance and inclusion reasons. Structural preparation does not start sessions, execute project checks or establish runtime readiness. The [project workflow](docs/project-workflow.md) explains manifests, dependencies, runtime profiles and run configuration.

   Run Apply only within an authorized bounded dispatch: `go run ./cmd/markitect apply --repo . --runtime-profile ABSOLUTE_RUNTIME_YAML --run-config ABSOLUTE_RUN_JSON` over a committed repository on a non-protected branch. It starts real Managers and needs a container engine and normal account credentials. The JSON report names its retained `evidencePath` under `<state>/<repositoryIdentity>/applies/`; closed Run records live under the sibling `runs/` directory. No question crosses Root to the invoking AI or the user; a material model blocker ends execution without publication. Apply never creates an owner commit.
3. Implementation-only work needs no intent change. A new artifact needs one exact registration in its owner's registry shard, with existing Definition links and a reason. An exceptional ignored file needs an exact observed path and reason in `.markitect/ignore.yaml`; Gitignore does not supply accounting. Canonical and control inputs cannot be ignored or registered as artifacts.
4. After changing `.markitect/model/`, preview the official readable view and write it using the returned `view.previewDigest`:

   ```text
   go run ./cmd/markitect docs --repo .
   go run ./cmd/markitect docs --repo . --expect PREVIEW_DIGEST --write
   ```

   Do not edit generated views. The model digest, rendered `contentDigest` and source/target-bound `previewDigest` have different meanings. Changed sources or targets require a fresh preview.
5. Before a PR, run the relevant local checks in [CONTRIBUTING](CONTRIBUTING.md#verify-a-change) and obtain an independent complete-diff review. Freeze the reviewed head and follow the [integration policy](docs/implementation-plan.md#integration). Source tests and smokes are bounded evidence, not provider, Apply, release or owner acceptance.

## Where things are

- [V1 concept 1.0](docs/concepts/v1-concept.md): the target.
- [Delivery plan](docs/implementation-plan.md): current steps, work streams, known deviations and integration rules.
- [V1 execution architecture](docs/design/v1-execution-architecture.md): concrete packages and contracts.
- [V1 qualification record](docs/validation/v1-qualification-20261024.md): native and source evidence so far, with its limits.
- [Code map](docs/development/code-map.md): every Go package and its owning guide.
- [Generated model view](docs/markitect/project.md): readable canonical Definitions.
- [Concept register](docs/concepts/register.md): accepted decisions and open questions.
- [Documentation map](docs/README.md): other documents.
