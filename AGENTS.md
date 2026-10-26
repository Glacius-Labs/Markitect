# Markitect agent entrypoint

Markitect is model-first development with delegated realization. People maintain a canonical model; a Manager hierarchy derived from it realizes, reviews and integrates the artifacts, and the host keeps a few clear technical guarantees.

**Current target:** [V1 concept 1.0](docs/concepts/v1-concept.md) ([DEC-068](docs/concepts/register.md#dec-068-v1-concept-10)). V1 is Model → Artifacts: Assessment, Implement and Verify each run through the complete model-derived hierarchy in fresh Run environments; Apply publishes only a stored, verified candidate after every Run has closed. Extract, Cockpit/UI, interactive authoring and Setup are not part of V1. V1 acceptance covers Codex with normal account login. The [delivery plan](docs/implementation-plan.md) owns the current steps and the known deviations of the code from the concept.

Before assessing product direction, read [Markitect in brief](docs/vision.md#markitect-in-brief), [Founding intent](docs/vision.md#founding-intent), [Common misreadings](docs/vision.md#common-misreadings) and the [concept register](docs/concepts/register.md). Do not reopen an accepted entry without new evidence. The canonical model is meant as a typed language in prose: projects define their own Kinds, Libraries supply common ones, the compiler checks structure only, and a model is judged like code.

## Working rules

- Writing chats run in parallel as far as is useful and their file scopes are disjoint ([DEC-070](docs/concepts/register.md#dec-070-parallel-writers-where-useful)), each with its own worktree and exclusive file scope. One writer at a time for shared contracts and generated files such as the code map. Codex development chats use Sol High, as [approved](docs/concepts/sources/v1-source-implementation-approval-20261023.md).
- Real container, provider or Apply runs need a concrete bounded dispatch under the [standing evidence authorization](docs/concepts/sources/v1-source-implementation-approval-20261023.md#subsequent-standing-evidence-authorization): source, limits, abort, cleanup and actual result. Failed attempts stay recorded. No release or deployment.
- Fix host/agent interaction failures in host state or the tool contract, not with more prompt text ([concept §3.7](docs/concepts/v1-concept.md#37-tasks-not-process-d13)).
- Write all documentation in English ([DEC-069](docs/concepts/register.md#dec-069-documentation-is-written-in-english)). Verbatim owner statements in `docs/concepts/sources/` keep their original wording (record rule; see the DEC-069 scope notes).

## Every task

Markitect itself is developed as an ordinary software project: agentic coding, tests and ordinary documentation, without a canonical self-model ([DEC-076](docs/concepts/register.md#dec-076-markitect-is-developed-as-a-regular-software-project)). Target projects keep `.markitect/`; product code and tests that handle it use fixtures.

1. Read the [working method](docs/development/working-method.md), the [delivery plan](docs/implementation-plan.md) and your bounded assignment. Use an isolated worktree from its integrated base and a non-protected branch with a scope prefix such as `dev/`, `fix/` or `docs/`. Verify path, branch, full SHA and local changes before writing. Preserve other sessions' work.
2. Find the owner of the change before writing: the package in the [code map](docs/development/code-map.md) with its owning guide, and the one canonical page for the topic ([documentation maintenance](docs/development/documentation.md)). Change behavior and its tests together.
3. Keep the documentation current in the same change: update the canonical page when a rule, command or boundary changes, and link to it instead of repeating it. A new or removed Go package also needs its [code map](docs/development/code-map.md) entry and import rule ([modules](docs/development/modules.md#adding-changing-or-removing-a-capability-package)). Do not edit generated pages; regenerate them.
4. Run Apply only within an authorized bounded dispatch and against a target project repository: `go run ./cmd/markitect apply --repo TARGET --runtime-profile ABSOLUTE_RUNTIME_YAML --run-config ABSOLUTE_RUN_JSON` over a committed target on a non-protected branch. It starts real Managers and needs a container engine and normal account credentials. The JSON report names its retained `evidencePath` under `<state>/<repositoryIdentity>/applies/`; closed Run records live under the sibling `runs/` directory. No question crosses Root to the invoking AI or the user; a material model blocker ends execution without publication. Apply never creates an owner commit. The [project workflow](docs/guides/project-workflow.md) explains manifests, dependencies, runtime profiles and run configuration.
5. Before a PR, run the relevant local checks in [CONTRIBUTING](CONTRIBUTING.md#verify-a-change) and obtain an independent complete-diff review. Freeze the reviewed head and follow the [integration policy](docs/implementation-plan.md#integration). Source tests and smokes are bounded evidence, not provider, Apply, release or owner acceptance.

## Where things are

- [V1 concept 1.0](docs/concepts/v1-concept.md): the target.
- [Delivery plan](docs/implementation-plan.md): current steps, work streams, known deviations and integration rules.
- [V1 execution architecture](docs/design/v1-execution-architecture.md): concrete packages and contracts.
- [V1 qualification record](docs/validation/v1-qualification-20261024.md): native and source evidence so far, with its limits.
- [Code map](docs/development/code-map.md): every Go package and its owning guide.
- [Concept register](docs/concepts/register.md): accepted decisions and open questions.
- [Documentation map](docs/README.md): other documents.
