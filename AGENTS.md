# Markitect agent entrypoint

Markitect is a programming language for documentation, with delegated realization ([DEC-101](docs/concepts/register.md#dec-101-markitect-is-a-programming-language-for-documentation)). People maintain one typed spec, the project's documentation and authority (code and older documents say model). The `markitect` tool checks and analyses it like code and records what was verified; agents started by the user's own Claude Code session realize it and verify it, and the tool keeps a few clear technical guarantees.

**Read these, in this order, before assessing or changing product direction:**

1. [Markitect in brief](docs/vision.md#markitect-in-brief), [Founding intent](docs/vision.md#founding-intent) and [Common misreadings](docs/vision.md#common-misreadings).
2. [Concept 4.0](docs/concepts/v4-concept.md), the one description of the product, which Main builds ([DEC-098](docs/concepts/register.md#dec-098-concept-40-is-the-one-concept)).
3. The [register](docs/concepts/register.md): the decisions in force, about thirty of them. Do not reopen one without new evidence.

Everything else that was decided or tried on the way, earlier concepts and plans included, is in the [archive](docs/archive.md). The spec is meant as a typed language in prose: projects define their own Kinds, Libraries supply common ones, a link is a declared dependency like an import, the compiler checks structure only, and a spec is judged like code ([what makes a good spec](docs/guides/good-spec.md)).

## Working rules

- Writing sessions run in parallel as far as is useful and their file scopes are disjoint ([DEC-070](docs/concepts/register.md#dec-070-parallel-writers-where-useful)), each with its own worktree and exclusive file scope. One writer at a time for shared contracts and generated files such as the code map. Codex development chats use Sol High, as [approved](docs/concepts/sources/v1-source-implementation-approval-20261023.md).
- Real agent runs of the skills, and any container or provider run, need a concrete bounded dispatch under the [standing evidence authorization](docs/concepts/sources/v1-source-implementation-approval-20261023.md#subsequent-standing-evidence-authorization): source, limits, abort, cleanup and actual result. Failed attempts stay recorded. A release follows the [release operations](packaging/README.md#release-operations); the owner allowed releasing on 6 November 2026 ([DEC-095](docs/concepts/register.md#dec-095-releases-a-plugin-and-a-binary-that-updates-itself)). No deployment.
- Fix agent interaction failures in the tool contract, not with more prompt text ([DEC-051](docs/concepts/register.md#dec-051-agents-get-tasks-not-process)).
- Write all documentation in English ([DEC-069](docs/concepts/register.md#dec-069-documentation-is-written-in-english)). Verbatim owner statements in `docs/concepts/sources/` keep their original wording.
- Main holds only what holds: history leaves Main through the [archive](docs/archive.md) in one change ([DEC-071](docs/concepts/register.md#dec-071-history-leaves-main-through-the-archive)).

## Every task

Markitect itself is developed as an ordinary software project: agentic coding, tests and ordinary documentation, without a spec of itself ([DEC-076](docs/concepts/register.md#dec-076-markitect-is-developed-as-an-ordinary-software-project)). Target projects keep `.markitect/`; product code and tests that handle it use fixtures.

1. Read the [working method](docs/development/working-method.md), the [delivery plan](docs/implementation-plan.md) and your bounded assignment. Use an isolated worktree from its integrated base and a non-protected branch with a scope prefix such as `dev/`, `fix/` or `docs/`. Verify path, branch, full SHA and local changes before writing. Preserve other sessions' work.
2. Find the owner of the change before writing: the package in the [code map](docs/development/code-map.md) with its owning guide, and the one canonical page for the topic ([documentation maintenance](docs/development/documentation.md)). Change behavior and its tests together.
3. Keep the documentation current in the same change: update the canonical page when a rule, command or boundary changes, and link to it instead of repeating it. A new owner decision gets an entry in the [register](docs/concepts/register.md), and concept 4.0 changes with it. A new, moved or removed Go package also changes its row in the [architecture rule table](internal/tooling/architecture/rules.go), which drives the import gate, the [code map](docs/development/code-map.md) and the source packager. Do not edit generated pages; regenerate them.
4. `/markitect-build`, `/markitect-review`, `/markitect-cleanup`, `/markitect-bughunt`, `/markitect-audit` and the routines lead to real agent work. Run them only within an authorized bounded dispatch and against a committed target project. The user's own Claude Code session starts the agent in that folder or in a worktree of it, with no Docker, token or background session ([project workflow](docs/guides/project-workflow.md#7-build-and-review)). The tool's commands start no agent and run anywhere.
5. Before a PR, run the relevant local checks in [CONTRIBUTING](CONTRIBUTING.md#verify-a-change) and obtain an independent complete-diff review. Freeze the reviewed head and follow the [integration policy](docs/implementation-plan.md#integration). Source tests and smokes are bounded evidence, not provider, agent-run, release or owner acceptance.

## Where things are

- [Vision](docs/vision.md): what Markitect is, why, where its trust comes from and what it leaves to people.
- [Concept 4.0](docs/concepts/v4-concept.md): the product, which Main builds.
- [Register](docs/concepts/register.md): the decisions in force, promises, assumptions, open questions and ideas.
- [Model language](docs/guides/model-language.md): the spec language, the `.markitect/` layout and the commands; [what makes a good spec](docs/guides/good-spec.md) says how to design one; the [project workflow](docs/guides/project-workflow.md) says how to use Markitect.
- [Delivery plan](docs/implementation-plan.md): current steps, work streams and integration rules.
- [Architecture](docs/architecture.md) and [code map](docs/development/code-map.md): packages, layers and every Go package with its owning guide.
- [Validation records](docs/validation/README.md): dated evidence for the current product, each with its limits.
- [Archive](docs/archive.md): earlier concepts, plans, decisions, designs and records.
- [Documentation map](docs/README.md): everything else.
