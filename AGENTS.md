# Markitect agent entrypoint

Markitect is a programming language for documentation, with delegated realization ([DEC-101](docs/decisions.md#dec-101-markitect-is-a-programming-language-for-documentation)). People maintain one typed spec of their project. The `markitect` tool checks and analyses it like code and records what was verified; agents started by the user's own Claude Code session realize it and verify it.

**Before you assess or change the product's direction,** read, in this order:

1. The [vision](docs/vision.md): [Markitect in brief](docs/vision.md#markitect-in-brief), [Founding intent](docs/vision.md#founding-intent) and [Common misreadings](docs/vision.md#common-misreadings).
2. The [concept](docs/concept.md), the one description of the product, which Main builds.
3. The [decisions](docs/decisions.md). Do not reopen one without new evidence.

**For every task,** follow [CONTRIBUTING](CONTRIBUTING.md): a worktree and branch of your own from current `origin/main`, the page that owns the topic, code, tests and documentation changed together, the checks, one independent review, and a merge of the reviewed head.

**Easy to miss:**

- Markitect itself has no spec. `.markitect/` in code and tests belongs to a target project and is handled with fixtures.
- Fix a failure in how agents work with Markitect in the tool contract, not with more prompt text ([DEC-051](docs/decisions.md#dec-051-agents-get-tasks-not-process)).
- `/markitect-build`, `/markitect-review`, `/markitect-cleanup`, `/markitect-bughunt`, `/markitect-audit` and the routines start real agents: run them only on a committed test project, bounded, as CONTRIBUTING says.
- Documentation is in English and describes only the current state, with one concept and no quotes.

[CONTRIBUTING](CONTRIBUTING.md#where-things-go) names the page for every topic, and the [roadmap](docs/roadmap.md) the current steps.
