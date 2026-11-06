# Markitect agent entrypoint

Markitect is spec-first development with delegated realization. People maintain one spec, the project's documentation and authority (code and older documents say canonical model); a Command has it realized by one implementer and one fresh reviewer, and the tool keeps a few clear technical guarantees.

**Main carries the build of [draft concept 4.0](docs/concepts/v4-concept.md)** since 6 November 2026, for the owner to try by using it ([DEC-094](docs/concepts/register.md#dec-094-main-carries-the-native-version-and-the-app-goes)); the draft is not accepted. `apply` and `verify` open a run in the project's own folder, and the user's own Claude Code session starts its builder and reviewer agents. Main until then, with the concept 3.0 rebuild and Markitect Desktop, is kept on the branch `archive/main-concept-3-20261106`.

**Accepted target:** [concept 3.0](docs/concepts/v3-concept.md) ([DEC-091](docs/concepts/register.md#dec-091-concept-30-is-the-target)), accepted on 4 November 2026; whether draft 4.0 replaces it is open. [V1 concept 1.0](docs/concepts/v1-concept.md) remains the record of V1. The [delivery plan](docs/implementation-plan.md) owns the current steps.

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
4. `apply` and `verify` lead to real agent work. Run them only within an authorized bounded dispatch and against a committed target project repository. On this branch they open a run, and the user's own Claude Code session starts the builder or reviewer agent in that folder, with no Docker, token or background session ([project workflow](docs/guides/project-workflow.md#7-apply-and-verify)). The model commands start no agent and run anywhere.
5. Before a PR, run the relevant local checks in [CONTRIBUTING](CONTRIBUTING.md#verify-a-change) and obtain an independent complete-diff review. Freeze the reviewed head and follow the [integration policy](docs/implementation-plan.md#integration). Source tests and smokes are bounded evidence, not provider, Apply, release or owner acceptance.

## Where things are

- [Draft concept 4.0](docs/concepts/v4-concept.md): what Main implements.
- [Concept 3.0](docs/concepts/v3-concept.md): the accepted target.
- [Concept 2.0](docs/concepts/v2-concept.md): the record of the Manager tree.
- [V1 concept 1.0](docs/concepts/v1-concept.md): the record of V1, which Main implemented until T1.
- [Model language](docs/guides/model-language.md): the 2.0 model language, the `.markitect/` layout and the model commands.
- [Delivery plan](docs/implementation-plan.md): current steps, work streams, known deviations and integration rules.
- [Architecture](docs/architecture.md): packages, layers and the product flow.
- [V1 qualification record](docs/validation/v1-qualification-20261024.md): native and source evidence so far, with its limits.
- [Code map](docs/development/code-map.md): every Go package and its owning guide.
- [Concept register](docs/concepts/register.md): accepted decisions and open questions.
- [Documentation map](docs/README.md): other documents.
