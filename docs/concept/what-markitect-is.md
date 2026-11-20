# What Markitect is

**The spec is a typed, declarative specification language in Markdown** ([DEC-119](../decisions.md#dec-119-people-architect-their-project-in-a-typed-declarative-specification-language)). It works like declarative programming, with an agent as the executor: it states what should hold, not how to get there, and agents make the project match it. It is a specification language as TLA+ and Alloy are, and nothing in it is executed. Its structure is formal and checked by the tool; its content is prose that an agent interprets. Because an agent can be wrong, every result is checked, and how well the checking works is measured ([§7](trust.md)).

## Terraform is the image

Terraform makes infrastructure match a configuration, keeps a state of what exists, and shows a plan of what would change. Markitect has the same three things, for the intent of a project ([DEC-124](../decisions.md#dec-124-a-state-maps-each-document-to-its-artifacts-and-status-is-the-plan)):

| Terraform | Markitect | What it is |
|---|---|---|
| Configuration | The spec | What should hold, like an interface ([§4](what-goes-in-a-spec.md)) |
| State | The state in `.markitect/` | Which artifacts realize which document, and their hashes at the last confirmation ([§6](the-state.md)) |
| `validate` | `check` | Is the spec correct in itself? |
| `plan` | `status` | What moved since the last confirmation? |
| `apply` | The build job | A builder realizes, a reviewer confirms ([§8](the-factory.md)) |
| Drift, kept by changing the configuration or reverted | A change past the spec | Back-filled into the spec or reverted ([§9](reports-and-actions.md#a-change-past-the-spec)) |
| The command line, pipeline definitions and the services that run them | The tool, the jobs and the runners | Computing, saying what each job does, and running it ([§11](the-tool.md), [§12](jobs-and-runners.md)) |
| `required_version` | The version in the project | Every runner uses the same tool |

A compiler would be the other image. Terraform fits better, because the work is reconciling what should be with what is, not translating one into the other. For readers without a developer's background, `status` is a checklist, computed instead of remembered: a team that knows every step can still skip one.

## The parts

| Part | What it is | Tied to a provider? |
|---|---|---|
| **Spec** | A folder of Markdown documents in the project, by default `docs/spec/` ([§3](the-language.md)) | No |
| **Tool** | One program, `markitect`. It checks the spec, computes the state and what is to verify, binds verdicts and computes the factory's figures. It starts no agent and uses no Git ([§11](the-tool.md)) | No |
| **Jobs** | For each job, which roles run in which order, with which task and write area. They ship in the Claude Code plugin ([§12](jobs-and-runners.md)) | Yes |
| **Runners** | Where a job runs: the user's own session, a scheduled task, a cloud task or CI. The user chooses ([§12](jobs-and-runners.md)) | Each runner is its own |

## Words

| Word | Meaning |
|---|---|
| **Spec** | The one description of the project that has authority over its artifacts |
| **Document, type** | One Markdown file of the spec, and a document whose sections are the template of other documents |
| **Package, template** | A folder of types, such as `software`; and a folder of documents with empty sections, applied to start an area |
| **Area** | A folder of the spec with a README of an area type, whose sections are its table of contents |
| **Artifact** | Any file that comes from the spec: code, tests, configuration, a handbook |
| **State** | One row per document in `.markitect/`: its artifacts, its counter-checks, its verdict and the hashes at its last confirmation |
| **History** | The facts recorded at each confirmation and move, as hashes, in `.markitect/` |
| **Report, action** | A signal that a change is needed, as a document in `.markitect/reports/`; and the branch that answers it |
| **Conversation, job** | Work with the owner; and every other run, from a commit to a branch, with nobody asked |
| **Runner** | Whatever starts a job and makes the Git steps |
| **Owner** | The person who owns the project and its spec: decides intent and makes the new decisions that are not handed over |
| **Architect, builder, reviewer** | The roles that write the spec, realize it, and confirm the realization |
| **Reader, counter-checker, sampler** | The roles that read and question the spec, derive checks from it alone, and plant known deviations in a throwaway copy |
| **Counter-check** | A test or a question with an expected answer, derived from the spec alone |
| **Control sample, spot check** | A run that measures how much the reviewer finds; and a fresh check of confirmed documents at random |
| **Landing rule** | The one rule that decides whether a job's branch lands |
| **Write table, control files** | What each role may write; and the files that steer the jobs, read from the target branch |

## Authority

Where an artifact differs from the spec, the artifact is wrong: code, tests, documentation, pipelines and configuration alike ([DEC-122](../decisions.md#dec-122-the-spec-declares-what-is-fixed-like-an-interface)). A project rule found outside the spec, in configuration or agent instructions, belongs in it. The roles that build never change the spec, and the architect writes no code. Where the spec leaves open something someone will rely on, nothing is guessed: it becomes a report ([§9](reports-and-actions.md)).

## Removing Markitect leaves an ordinary repository

Nothing of Markitect stands in an artifact or in the spec: no comment in code, no line in a handbook, no front matter. The spec is ordinary documentation, and everything else of Markitect lies in `.markitect/`. Deleting `.markitect/` and the plugin leaves a repository with its documentation, code and tests, the counter-checks included ([DEC-133](../decisions.md#dec-133-the-project-holds-only-the-spec-and-markitect-and-pins-its-version)).
