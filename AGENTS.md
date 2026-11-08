# Markitect agent entrypoint

Markitect is spec-first development with delegated realization. People maintain one spec, the project's documentation and authority (code and older documents say canonical model); a Command has it realized by one implementer and one fresh reviewer, and the tool keeps a few clear technical guarantees.

**Accepted target: [concept 4.0](docs/concepts/v4-concept.md)** ([DEC-098](docs/concepts/register.md#dec-098-concept-40-is-the-target)), accepted on 7 November 2026; Main carries its build since 6 November 2026 ([DEC-094](docs/concepts/register.md#dec-094-main-carries-the-native-version-and-the-app-goes)). The skills, named after the activity (`markitect-design`, `/markitect-build`, `/markitect-review`, `/markitect-cleanup`, `/markitect-bughunt`, `/markitect-audit`, `/markitect-schedule`, `/markitect-config`), have the user's own Claude Code session start the builder, reviewer, architect or auditor agent, whose rights a hook holds; the builder works in a worktree of its own, and a long list is worked in parallel per Package through a shipped workflow, by a strategy that `.markitect/harness.yaml` or the user's `~/.markitect/harness.yaml` sets ([DEC-102](docs/concepts/register.md#dec-102-claude-chooses-the-parallelism-per-run-through-a-shipped-workflow)). `markitect status` says what is left to verify, the reviewer records each verdict with `markitect mark` in `.markitect/review.sum`, and the tool steers no work and uses no Git ([DEC-099](docs/concepts/register.md#dec-099-markitect-is-a-tool-with-a-verification-cache), [DEC-100](docs/concepts/register.md#dec-100-skills-are-named-after-the-activity-agents-after-the-role), [tooling plan](docs/concepts/tooling-plan.md)). Main until then, with the concept 3.0 rebuild and Markitect Desktop, is kept on the branch `archive/main-concept-3-20261106`.

**Records:** [concept 3.0](docs/concepts/v3-concept.md) ([DEC-091](docs/concepts/register.md#dec-091-concept-30-is-the-target)) was the target from 4 to 7 November 2026, and [V1 concept 1.0](docs/concepts/v1-concept.md) remains the record of V1. The [delivery plan](docs/implementation-plan.md) owns the current steps.

Before assessing product direction, read [Markitect in brief](docs/vision.md#markitect-in-brief), [Founding intent](docs/vision.md#founding-intent), [Common misreadings](docs/vision.md#common-misreadings) and the [concept register](docs/concepts/register.md). Do not reopen an accepted entry without new evidence. The canonical model is meant as a typed language in prose: projects define their own Kinds, Libraries supply common ones, the compiler checks structure only, and a model is judged like code.

## Working rules

- Writing chats run in parallel as far as is useful and their file scopes are disjoint ([DEC-070](docs/concepts/register.md#dec-070-parallel-writers-where-useful)), each with its own worktree and exclusive file scope. One writer at a time for shared contracts and generated files such as the code map. Codex development chats use Sol High, as [approved](docs/concepts/sources/v1-source-implementation-approval-20261023.md).
- Real container, provider or Apply runs need a concrete bounded dispatch under the [standing evidence authorization](docs/concepts/sources/v1-source-implementation-approval-20261023.md#subsequent-standing-evidence-authorization): source, limits, abort, cleanup and actual result. Failed attempts stay recorded. A release follows the [release operations](packaging/README.md#release-operations); the owner allowed releasing on 6 November 2026 ([DEC-095](docs/concepts/register.md#dec-095-releases-return-with-a-plugin-and-a-binary-that-updates-itself)). No deployment.
- Fix host/agent interaction failures in host state or the tool contract, not with more prompt text ([concept §3.7](docs/concepts/v1-concept.md#37-tasks-not-process-d13)).
- Write all documentation in English ([DEC-069](docs/concepts/register.md#dec-069-documentation-is-written-in-english)). Verbatim owner statements in `docs/concepts/sources/` keep their original wording (record rule; see the DEC-069 scope notes).

## Every task

Markitect itself is developed as an ordinary software project: agentic coding, tests and ordinary documentation, without a canonical self-model ([DEC-076](docs/concepts/register.md#dec-076-markitect-is-developed-as-a-regular-software-project)). Target projects keep `.markitect/`; product code and tests that handle it use fixtures.

1. Read the [working method](docs/development/working-method.md), the [delivery plan](docs/implementation-plan.md) and your bounded assignment. Use an isolated worktree from its integrated base and a non-protected branch with a scope prefix such as `dev/`, `fix/` or `docs/`. Verify path, branch, full SHA and local changes before writing. Preserve other sessions' work.
2. Find the owner of the change before writing: the package in the [code map](docs/development/code-map.md) with its owning guide, and the one canonical page for the topic ([documentation maintenance](docs/development/documentation.md)). Change behavior and its tests together.
3. Keep the documentation current in the same change: update the canonical page when a rule, command or boundary changes, and link to it instead of repeating it. A new, moved or removed Go package also changes its row in the [architecture rule table](internal/tooling/architecture/rules.go), which drives the import gate, the [code map](docs/development/code-map.md) and the source packager. Do not edit generated pages; regenerate them.
4. `/markitect-build`, `/markitect-review`, `/markitect-cleanup`, `/markitect-bughunt`, `/markitect-audit` and the routines lead to real agent work. Run them only within an authorized bounded dispatch and against a committed target project. The user's own Claude Code session starts the agent in that folder or in a worktree of it, with no Docker, token or background session ([project workflow](docs/guides/project-workflow.md#7-build-and-review)). The tool's commands start no agent and run anywhere.
5. Before a PR, run the relevant local checks in [CONTRIBUTING](CONTRIBUTING.md#verify-a-change) and obtain an independent complete-diff review. Freeze the reviewed head and follow the [integration policy](docs/implementation-plan.md#integration). Source tests and smokes are bounded evidence, not provider, Apply, release or owner acceptance.

## Where things are

- [Concept 4.0](docs/concepts/v4-concept.md): the accepted target, which Main implements.
- [Concept 3.0](docs/concepts/v3-concept.md): the record of the target from 4 to 7 November 2026.
- [Concept 2.0](docs/concepts/v2-concept.md): the record of the Manager tree.
- [V1 concept 1.0](docs/concepts/v1-concept.md): the record of V1, which Main implemented until T1.
- [Model language](docs/guides/model-language.md): the 2.0 model language, the `.markitect/` layout and the model commands.
- [Delivery plan](docs/implementation-plan.md): current steps, work streams, known deviations and integration rules.
- [Architecture](docs/architecture.md): packages, layers and the product flow.
- [V1 qualification record](docs/validation/v1-qualification-20261024.md): native and source evidence so far, with its limits.
- [Code map](docs/development/code-map.md): every Go package and its owning guide.
- [Concept register](docs/concepts/register.md): accepted decisions and open questions.
- [Documentation map](docs/README.md): other documents.
