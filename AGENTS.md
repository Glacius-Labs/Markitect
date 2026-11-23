# Markitect agent entrypoint

With Markitect, people architect their project in a typed, declarative specification language in Markdown ([DEC-119](docs/decisions.md#dec-119-people-architect-their-project-in-a-typed-declarative-specification-language)). They maintain one spec of it, its way of working included, which is the project's own. Markitect is the language, the default that ships with it, today the one module `core` of general types, and the `markitect` tool, which checks the spec, analyses it and helps reading it; the tool never writes and keeps no state ([DEC-154](docs/decisions.md#dec-154-markitect-is-a-language-its-default-and-a-tool-that-only-reads), [roadmap](docs/roadmap.md#where-main-is)).

**Before you assess or change the product's direction,** read, in this order:

1. The [vision](docs/vision.md): [Markitect in brief](docs/vision.md#markitect-in-brief), [Why Markitect exists](docs/vision.md#why-markitect-exists), [Founding intent](docs/vision.md#founding-intent) and [Common misreadings](docs/vision.md#common-misreadings).
2. The [concept](docs/concept/README.md), the one description of the product, which Main builds.
3. The [decisions](docs/decisions.md). Do not reopen one without new evidence.

**For every task,** follow [CONTRIBUTING](CONTRIBUTING.md): a worktree and branch of your own from current `origin/main`, the page that owns the topic, code, tests and documentation changed together, the checks, one independent review, and a merge of the reviewed head.

**Easy to miss:**

- Markitect itself has no spec. `.markitect/` (its files `spec` and `ignore`) in code and tests belongs to a target project and is handled with fixtures.
- Documentation is in English and describes only the current state, with one concept and no quotes.

[CONTRIBUTING](CONTRIBUTING.md#where-things-go) names the page for every topic, and the [roadmap](docs/roadmap.md) what comes next.
