# Markitect agent entrypoint

With Markitect, people architect their project in a typed, declarative specification language in Markdown ([DEC-119](docs/decisions.md#dec-119-people-architect-their-project-in-a-typed-declarative-specification-language)). They maintain one spec of it, its way of working included: a circle for a team, which skills in the repository realize. The `markitect` tool checks the spec and computes what is built and verified; it steers nothing. The tool and the skills of the standard are built on Main; no skill has run on a test project yet ([roadmap](docs/roadmap.md#where-main-is)).

**Before you assess or change the product's direction,** read, in this order:

1. The [vision](docs/vision.md): [Markitect in brief](docs/vision.md#markitect-in-brief), [Why Markitect exists](docs/vision.md#why-markitect-exists), [Founding intent](docs/vision.md#founding-intent) and [Common misreadings](docs/vision.md#common-misreadings).
2. The [concept](docs/concept/README.md), the one description of the product, which Main builds.
3. The [decisions](docs/decisions.md). Do not reopen one without new evidence.

**For every task,** follow [CONTRIBUTING](CONTRIBUTING.md): a worktree and branch of your own from current `origin/main`, the page that owns the topic, code, tests and documentation changed together, the checks, one independent review, and a merge of the reviewed head.

**Easy to miss:**

- Markitect itself has no spec. `.markitect/` in code and tests belongs to a target project and is handled with fixtures.
- A failure in how agents work with Markitect is fixed in the way of working, the `factory` template, and the skill built from it, not with prompt text beside them ([DEC-147](docs/decisions.md#dec-147-the-way-of-working-stands-in-the-spec-and-skills-realize-it)).
- The skills of the standard, `norman`, `capture`, `architect`, `finish-planning`, `build`, `review`, `analyze`, `accept`, `observe`, `create-observer`, `triage`, `record`, `brief` and `housekeeper`, start real agents in a project set up with `init`: run them only on a committed test project, bounded, with the product owner's word, as CONTRIBUTING says.
- Documentation is in English and describes only the current state, with one concept and no quotes.

[CONTRIBUTING](CONTRIBUTING.md#where-things-go) names the page for every topic, and the [roadmap](docs/roadmap.md) the current steps.
