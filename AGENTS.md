# Markitect agent entrypoint

Markitect is spec-first development with delegated realization. People maintain one spec, the project's documentation and authority (code and older documents say canonical model). How agents realize it is being redesigned.

**Current state:** no concept is the target for realization ([DEC-094](docs/concepts/register.md#dec-094-a-fresh-start-for-realization)). Main holds the model language, the model commands, the skills and Markitect Desktop. `apply`, `verify` and the evaluation suite were removed for a fresh start and stay at the [archive point](docs/archive.md). The aim is one clean and simple version that works reliably on Sonnet, no longer proof that one version is better. The [delivery plan](docs/implementation-plan.md) owns the current steps.

Before assessing product direction, read [Markitect in brief](docs/vision.md#markitect-in-brief), [Founding intent](docs/vision.md#founding-intent), [Common misreadings](docs/vision.md#common-misreadings) and the [concept register](docs/concepts/register.md). Do not reopen an accepted entry without new evidence. The canonical model is meant as a typed language in prose: projects define their own Kinds, Libraries supply common ones, the compiler checks structure only, and a model is judged like code.

## Working rules

- Writing chats run in parallel as far as is useful and their file scopes are disjoint ([DEC-070](docs/concepts/register.md#dec-070-parallel-writers-where-useful)), each with its own worktree and exclusive file scope. One writer at a time for shared contracts and generated files such as the code map. Codex development chats use Sol High, as [approved](docs/concepts/sources/v1-source-implementation-approval-20261023.md).
- Real container or provider runs need a concrete bounded dispatch under the [standing evidence authorization](docs/concepts/sources/v1-source-implementation-approval-20261023.md#subsequent-standing-evidence-authorization): source, limits, abort, cleanup and actual result. Failed attempts stay recorded. No release or deployment.
- Fix host/agent interaction failures in host state or the tool contract, not with more prompt text ([concept §3.7](docs/concepts/v1-concept.md#37-tasks-not-process-d13)).
- Write all documentation in English ([DEC-069](docs/concepts/register.md#dec-069-documentation-is-written-in-english)). Verbatim owner statements in `docs/concepts/sources/` keep their original wording (record rule; see the DEC-069 scope notes).

## Every task

Markitect itself is developed as an ordinary software project: agentic coding, tests and ordinary documentation, without a canonical self-model ([DEC-076](docs/concepts/register.md#dec-076-markitect-is-developed-as-a-regular-software-project)). Target projects keep `.markitect/`; product code and tests that handle it use fixtures.

1. Read the [working method](docs/development/working-method.md), the [delivery plan](docs/implementation-plan.md) and your bounded assignment. Use an isolated worktree from its integrated base and a non-protected branch with a scope prefix such as `dev/`, `fix/` or `docs/`. Verify path, branch, full SHA and local changes before writing. Preserve other sessions' work.
2. Find the owner of the change before writing: the package in the [code map](docs/development/code-map.md) with its owning guide, and the one canonical page for the topic ([documentation maintenance](docs/development/documentation.md)). Change behavior and its tests together.
3. Keep the documentation current in the same change: update the canonical page when a rule, command or boundary changes, and link to it instead of repeating it. A new, moved or removed Go package also changes its row in the [architecture rule table](internal/tooling/architecture/rules.go), which drives the import gate, the [code map](docs/development/code-map.md) and the source packager. Do not edit generated pages; regenerate them.
4. The model commands start no agent and run anywhere. The app's chats need Docker, an image and the user's own token from `claude setup-token` in `MARKITECT_CLAUDE_TOKEN`, never an API key.
5. Before a PR, run the relevant local checks in [CONTRIBUTING](CONTRIBUTING.md#verify-a-change) and obtain an independent complete-diff review. Freeze the reviewed head and follow the [integration policy](docs/implementation-plan.md#integration). Source tests and smokes are bounded evidence, not provider, release or owner acceptance.

## Where things are

- [Concept 3.0](docs/concepts/v3-concept.md), [concept 2.0](docs/concepts/v2-concept.md) and [V1 concept 1.0](docs/concepts/v1-concept.md): the record of earlier targets.
- [Model language](docs/guides/model-language.md): the model language, the `.markitect/` layout and the model commands.
- [Delivery plan](docs/implementation-plan.md): current steps, work streams and integration rules.
- [Architecture](docs/architecture.md): packages, layers and the product flow.
- [Code map](docs/development/code-map.md): every Go package and its owning guide.
- [Concept register](docs/concepts/register.md): accepted decisions and open questions.
- [Documentation map](docs/README.md): other documents.
