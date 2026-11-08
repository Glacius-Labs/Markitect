# Documentation map

Use this map to find the document that owns a subject. It routes to those owners instead of repeating them.

## Start by task

| If you want to… | Start here |
|---|---|
| Understand what Markitect is, before assessing or redirecting it | [Markitect in brief](vision.md#markitect-in-brief), [Common misreadings](vision.md#common-misreadings), then [concept 4.0](concepts/v4-concept.md) and the [decisions in force](concepts/register.md) |
| Use Markitect on a project: set it up, design and register the spec, check and vet it, and build, review, clean up, hunt bugs and audit with the skills | [Project workflow](guides/project-workflow.md), the one user guide |
| Write or read a spec: Definitions, Kinds and Libraries, links, the registry, the commands and their diagnostics | [Model language](guides/model-language.md), the canonical description of the language |
| Judge whether a spec is well designed | [What makes a good spec](guides/good-spec.md) |
| Implement Markitect | [Concept 4.0](concepts/v4-concept.md), the [delivery plan](implementation-plan.md), the [architecture](architecture.md) and the [code map](development/code-map.md) |
| Contribute source | [Contribution checks](../CONTRIBUTING.md), the [development documents](#development) below and the [repository layout](development/repository-layout.md) |
| Operate a release | The [distribution guide](../packaging/README.md#release-operations) |
| Find an earlier concept, plan, decision, design, experiment or dated record | The [archive index](archive.md) |

<a id="canonical-status-and-evidence-owners"></a>

## Canonical owners

This is the one owner table; other pages link here instead of keeping their own.

| Subject | Owner |
|---|---|
| What Markitect is and why, where its trust comes from, its origin and what it leaves to people | [Vision](vision.md) |
| The product as a whole: the spec, the tool, the record, the skills, the harness and how sure each guarantee is | [Concept 4.0](concepts/v4-concept.md) |
| The owner's decisions in force, promises, assumptions, enhancements, open questions and ideas | [Register](concepts/register.md), under the [record rules](concepts/README.md) |
| The spec language, the `.markitect/` layout and the commands | [Model language](guides/model-language.md) |
| Designing a good spec | [What makes a good spec](guides/good-spec.md) |
| Using Markitect on a project | [Project workflow](guides/project-workflow.md) |
| Technical behavior, package contracts and invariants | [Architecture](architecture.md), the package READMEs and the guides above |
| Current steps, work streams and integration | [Delivery plan](implementation-plan.md); actual candidates and checks live in PRs |
| Dated evidence for the current product | [Validation records](validation/README.md), read at their recorded scope and source |
| Published asset identity of the retired releases | [Distribution retirement record](validation/distribution-retirement-20261017.md) and its [retired release mapping](validation/retired-releases-20261017.json) |
| Repository layout and where new files go | [Repository layout](development/repository-layout.md) |
| Earlier concepts, plans, decisions, designs, experiments and records | [Archive index](archive.md) |
| Contribution and hosted validation commands, pull requests and parallel work | [CONTRIBUTING.md](../CONTRIBUTING.md) |

Markitect itself is developed as an ordinary software project without a spec of itself ([DEC-076](concepts/register.md#dec-076-markitect-is-developed-as-an-ordinary-software-project)); these documents are its canonical description, one owner per topic.

## Development

Start engineering work through [AGENTS.md](../AGENTS.md), the [working method](development/working-method.md) and the canonical owner of the change.

| Document | Covers |
|---|---|
| [Working method](development/working-method.md) | The owner-adopted practice of explicit learning, complete workflows, focused design review and short milestone accounts |
| [Documentation maintenance](development/documentation.md) | Placement, navigation, finding the canonical owner, and moving history to the archive |
| [Repository layout](development/repository-layout.md) | Where everything lives in this repository, where new things go and what enforces it |
| [Code map](development/code-map.md) | Every Go package with its layer, purpose and owning document; generated and checked by a test |
