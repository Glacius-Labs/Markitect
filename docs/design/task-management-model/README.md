# Task management: a complete documentation model

Status: proposed design example, prepared 15 October 2026. The owner requested a complete specification first. No task application is implemented here, and no application Check has been executed. The seven kinds are a modeling proposal, not the current Markitect schema.

## The system

A person uses a local command-line application to create tasks, list them and mark them completed. Tasks survive an ordinary restart. The first version has one local task file and no shared use, synchronization or reminders.

The proposed public commands are `tasks add "Buy milk"`, `tasks list` and `tasks complete 1`. These are commands of the future application, not commands that this branch supplies. The specification settles observable behavior while leaving the programming language and storage encoding to implementation.

## Reading order

1. [Original project description](source-description.md): the conversational input, preserved without silently adopting later refinements.
2. [Model language](model-language.md): the seven kinds, status axes and explicit relationship meanings.
3. [Concepts](concepts.md), [Requirements](requirements.md) and [Use Cases](use-cases.md): the domain, shared obligations and complete caller-visible behavior.
4. [Components](components.md) and [Artifacts](artifacts.md): responsibility boundaries and planned repository realization.
5. [Checks](checks.md) and [Decisions](decisions.md): expected verification and the reasons for choices or deliberate freedom.
6. [Relationships](relations.md): the canonical edge list; prose links are navigation.
7. [Boundary and completeness review](assessment.md): input coverage, pressure tests and remaining design questions for Markitect.

## Structure

```text
task-management-model/
  README.md                 reading order and status
  source-description.md     original input
  model-language.md         candidate vocabulary and relationship contract
  concepts.md               Concept records
  requirements.md           Requirement records
  use-cases.md              UseCase records
  components.md             Component records
  artifacts.md              Artifact records
  checks.md                 Check records
  decisions.md              Decision records
  relations.md              explicit edges
  assessment.md             coverage and boundary review
```

Each record has an explicit stable identity and a readable definition. One file per kind keeps this small example easy to inspect; it is not a universal storage layout. The whole package is a reviewable draft. Committing it does not accept the candidate types or the fictional product's decisions.

For current Markitect behavior, use the [architecture guide](../../architecture.md). Return to the [design map](../README.md).
