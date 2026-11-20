# Markitect agent entrypoint

With Markitect, people architect their project in a typed, declarative specification language in Markdown ([DEC-119](docs/decisions.md#dec-119-people-architect-their-project-in-a-typed-declarative-specification-language)). They maintain one spec of it. The `markitect` tool checks it and computes what is built and verified; jobs, started by a runner the user chooses, realize it and check it.

**Before you assess or change the product's direction,** read, in this order:

1. The [vision](docs/vision.md): [Markitect in brief](docs/vision.md#markitect-in-brief), [Why Markitect exists](docs/vision.md#why-markitect-exists), [Founding intent](docs/vision.md#founding-intent) and [Common misreadings](docs/vision.md#common-misreadings).
2. The [concept](docs/concept/README.md), the one description of the product, which Main builds.
3. The [decisions](docs/decisions.md). Do not reopen one without new evidence.

**For every task,** follow [CONTRIBUTING](CONTRIBUTING.md): a worktree and branch of your own from current `origin/main`, the page that owns the topic, code, tests and documentation changed together, the checks, one independent review, and a merge of the reviewed head.

**Easy to miss:**

- Markitect itself has no spec. `.markitect/` in code and tests belongs to a target project and is handled with fixtures.
- Fix a failure in how agents work with Markitect in the tool contract, not with more prompt text ([DEC-141](docs/decisions.md#dec-141-agents-get-tasks-not-process)).
- The code, the guides in `docs/guides/` and the pages under `internal/` are still the 0.34.0 tool of the earlier concept; the [roadmap](docs/roadmap.md#where-main-is) says how they are rebuilt to the concept.
- `/markitect-architect`, `/markitect-build`, `/markitect-review`, `/markitect-cleanup`, `/markitect-bughunt`, `/markitect-audit` and the routines start real agents: run them only on a committed test project, bounded, as CONTRIBUTING says.
- Documentation is in English and describes only the current state, with one concept and no quotes.

[CONTRIBUTING](CONTRIBUTING.md#where-things-go) names the page for every topic, and the [roadmap](docs/roadmap.md) the current steps.
