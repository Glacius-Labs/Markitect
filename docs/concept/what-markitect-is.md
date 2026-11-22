# What Markitect is

**The spec is a typed, declarative specification language in Markdown** ([DEC-119](../decisions.md#dec-119-people-architect-their-project-in-a-typed-declarative-specification-language)). It works like declarative programming, with an agent as the executor: it states what should hold, not how to get there, and agents make the project match it. It is a specification language as TLA+ and Alloy are, and nothing in it is executed. Its structure is formal and checked by the tool; its content is prose that an agent interprets. Because an agent can be wrong, every result is checked ([§7](trust.md)).

The spec describes the whole project, its way of working included. Around it, a team works in one circle of observing, triage, briefing, recording and work ([§8](the-circle.md)), and skills realize that way of working as code realizes the product.

## Terraform is the image

Terraform makes infrastructure match a configuration, keeps a state of what exists, and shows a plan of what would change. Markitect has the same three things, for the intent of a project ([DEC-124](../decisions.md#dec-124-a-state-maps-each-document-to-its-artifacts-and-status-is-the-plan)):

| Terraform | Markitect | What it is |
|---|---|---|
| Configuration | The spec | What should hold, like an interface ([§4](what-goes-in-a-spec.md)) |
| State | The state in `.markitect/` | Which artifacts realize which document, and their hashes at the last confirmation ([§6](the-state.md)) |
| `validate` | `check` | Is the spec correct in itself? |
| `plan` | `status` | What moved since the last confirmation? |
| `apply` | Building a story | A builder realizes, a reviewer confirms ([§8](the-circle.md)) |
| Drift | A change past the spec | `status` shows it, and it becomes a finding ([§9](the-work.md)) |
| Modules from the registry | Packages and templates from the library | Copied into the spec, which owns its copy ([§3](the-language.md)) |
| `required_version` | The version in the project | Everyone uses the same tool |

A compiler would be the other image. Terraform fits better, because the work is reconciling what should be with what is, not translating one into the other. For readers without a developer's background, `status` is a checklist, computed instead of remembered: a team that knows every step can still skip one.

## The parts

Markitect has three parts ([DEC-148](../decisions.md#dec-148-markitect-is-a-tool-a-spec-with-templates-and-skills)).

| Part | What it is | Tied to a provider? |
|---|---|---|
| **Tool** | One program, `markitect`. It checks the spec and the documents of the work, computes the state and what is to verify, and binds verdicts. It starts no agent, steers nothing and uses no Git ([§11](the-tool.md)) | No |
| **Spec with templates** | A folder of Markdown documents in the project, by default `docs/spec/`, started from four templates; the `factory` template holds the way of working as a standard ([§3](the-language.md), [§4](what-goes-in-a-spec.md)) | No |
| **Skills** | One per agent, built from the `factory` part of the spec and lying in the repository as its artifacts ([§12](the-project.md)) | No: they hold for Claude Code and Codex alike |

## Words

| Word | Meaning |
|---|---|
| **Spec** | The one description of the project that has authority over its artifacts: the target |
| **Document, type** | One Markdown file of the spec, and a document whose sections are the template of other documents |
| **Package, template** | A folder of types, such as `software`; and a folder of documents with empty sections, applied to start a part of the spec |
| **Part** | A folder of the spec, with a README as its table of contents |
| **Artifact** | Any file that comes from the spec: code, tests, configuration, a handbook, a skill. The artifacts are what is |
| **State** | One row per document in `.markitect/`: its artifacts, its verdict and the hashes at its last confirmation |
| **History** | The facts recorded at each confirmation and move, as hashes, in `.markitect/` |
| **The way of working, factory** | How the team works, and the part of the spec and the template that hold it |
| **The circle** | The stations of the way of working: Observe, Triage, Briefing, Record, Work |
| **Context** | The folder of everything that happened: reports and dropped material |
| **Report** | What an agent or observer writes about one run or session |
| **Finding** | One thing that deserves attention, without a solution |
| **Briefing** | What reaches people: short, ordered by importance |
| **Plan, plan item** | The tree of the work, rooted in the Areas of the spec: Feature, Story and Task; and a Feature or a Story of it, whose documents lie in `.markitect/plan/` |
| **Area, class** | An area of planning, a document of the spec in its `factory` part, with an owner and an optional class; and what a group of Areas is and which rules hold for it |
| **Target picture** | The section of a Feature or a Story that says what should hold once it is done |
| **Person, role** | Someone on the team; and Admin, Developer or QA, each with its usual way of working |
| **Agent** | A fresh session that runs exactly one skill |
| **Deviation** | Work outside the usual way of working, with a level; nothing is forbidden |
| **`confirm`** | What the reviewer writes: a verdict with its artifacts and evidence, bound to the state it read |
| **Counter-check, control sample, spot check** | A check derived from the spec alone; a run that measures how much the reviewer finds; a fresh check of confirmed documents. They come later, with the runs ([§7](trust.md)) |

## Authority

Where an artifact differs from the spec, the artifact is wrong: code, tests, documentation, pipelines, configuration and skills alike ([DEC-122](../decisions.md#dec-122-the-spec-declares-what-is-fixed-like-an-interface)). A project rule found outside the spec, in configuration or agent instructions, belongs in it. Where the spec leaves open something someone will rely on, nothing is guessed: it becomes a finding ([§9](the-work.md)).

## Removing Markitect leaves an ordinary repository

Nothing of Markitect stands in an artifact or in the spec: no comment in code, no line in a handbook, no front matter. The spec is ordinary documentation, the skills are ordinary skills of the project, and everything else of Markitect lies in `.markitect/`. Deleting `.markitect/` and the tool leaves a repository with its documentation, code, tests and skills ([DEC-149](../decisions.md#dec-149-markitect-holds-the-work-and-init-lays-out-templates-and-skills)).
