# Markitect agent entrypoint

Markitect is model-first development with delegated realization. People maintain a canonical model; a Manager hierarchy derived from it realizes, reviews and integrates the artifacts, and the host keeps a few clear technical guarantees.

**Current target:** [concept 2.0](docs/concepts/v2-concept.md) ([DEC-086](docs/concepts/register.md#dec-086-concept-20-is-the-target)), accepted on 28 October 2026: the model as typed documentation in Markdown, one Run per change with a maker and a checker per Manager, Git handoffs with server-side hooks, a native host. Main is being rebuilt to it in steps T1 to T5, with no compatibility kept. T1 built the model language and the commands `init`, `check`, `lock`, `refs`, `graph`, `impact` and `lint` ([model language](docs/guides/model-language.md)) and removed V1's Run host; no command starts an agent until T4 brings `apply` and `verify`. [V1 concept 1.0](docs/concepts/v1-concept.md) ([DEC-068](docs/concepts/register.md#dec-068-v1-concept-10)) and its code before T1 remain the record of V1. The [delivery plan](docs/implementation-plan.md) owns the current steps, the way from Main to concept 2.0 and the known deviations of the code.

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
3. Keep the documentation current in the same change: update the canonical page when a rule, command or boundary changes, and link to it instead of repeating it. A new, moved or removed Go package also changes its row in the [architecture rule table](internal/tooling/architecture/rules.go), which drives the import gate, the [code map](docs/development/code-map.md) and the source packager. Do not edit generated pages; regenerate them.
4. Main has no `apply` until T4 builds the Run engine. Once it exists, run it only within an authorized bounded dispatch and against a committed target project repository; it starts real Managers and needs a container engine and a normal account login, never an API key. The model commands start no agent and run anywhere ([project workflow](docs/guides/project-workflow.md)).
5. Before a PR, run the relevant local checks in [CONTRIBUTING](CONTRIBUTING.md#verify-a-change) and obtain an independent complete-diff review. Freeze the reviewed head and follow the [integration policy](docs/implementation-plan.md#integration). Source tests and smokes are bounded evidence, not provider, Apply, release or owner acceptance.

## Where things are

- [Concept 2.0](docs/concepts/v2-concept.md): the target.
- [V1 concept 1.0](docs/concepts/v1-concept.md): the record of V1, which Main implemented until T1.
- [Model language](docs/guides/model-language.md): the 2.0 model language, the `.markitect/` layout and the model commands.
- [Delivery plan](docs/implementation-plan.md): current steps, work streams, known deviations and integration rules.
- [Architecture](docs/architecture.md): packages, layers and the product flow.
- [V1 qualification record](docs/validation/v1-qualification-20261024.md): native and source evidence so far, with its limits.
- [Code map](docs/development/code-map.md): every Go package and its owning guide.
- [Concept register](docs/concepts/register.md): accepted decisions and open questions.
- [Documentation map](docs/README.md): other documents.
