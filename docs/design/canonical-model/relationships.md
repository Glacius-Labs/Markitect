# Relationships between the terms

Status: working design, illustrated by the [Tasks example](task-management-example.md).

The terms describe different aspects of a project. Their relationships explain how those aspects fit together. This page expresses those relationships in ordinary language; the examples give each connection a concrete meaning.

## Project and interaction

| Connection | Meaning | Tasks example |
|---|---|---|
| [Project](project.md) brings definitions together | Each definition belongs to the project it describes. | Tasks brings together its Actors, UseCases, Concepts and other definitions. |
| [Actor](actor.md) uses an [Interface](interface.md) | A role interacts with the software through an interaction point. | User uses the Tasks command line. |
| Interface exposes a [UseCase](use-case.md) | The interaction point makes an action available to an Actor. | The command line exposes Add task. |
| UseCase works with [Concepts](concept.md) | The action uses, creates or changes things described by the project's domain vocabulary. | Add task creates a Task with a Title and a TaskStatus. |
| Concept relates to another Concept | The relationship explains how domain terms fit together. | A Task has a Title and a TaskStatus. |

An Actor is a role in a stated context. User interacts with the application. Contributor and Reviewer participate in work on the project. A person can fulfil several roles. The relationship identifies what that role does in each context.

## Structure and files

| Connection | Meaning | Tasks example |
|---|---|---|
| [Component](component.md) contains Components | A system part can be described through its constituent parts. | The Tasks application contains Task Logic, Application, Command Line and Local Storage. |
| Component depends on another Component | One system part relies on another; the description states the kind of dependency. | Command Line uses Application to perform task operations. |
| Component handles a UseCase | A system part is responsible for carrying out an offered action. | Application handles Add task. |
| Component provides an Interface | A system part supplies the interaction point. | Command Line provides the Tasks command-line interface. |
| Component has associated [Artifacts](artifact.md) | Files express or support the system part; their purpose and location are stated. | Task Logic has source files, tests and a guide to task behavior. |

Composition and dependency express different relationships. Task Logic belongs to the Tasks application. Command Line depends on Application. A description names each relationship explicitly.

An Artifact can relate to several definitions. A guide can describe several UseCases; a test file can contain several Checks. Each connection states its purpose, such as implements, documents or verifies. A Rule can require a particular Artifact to accompany a Component.

## Rules, conditions and checks

| Connection | Meaning | Tasks example |
|---|---|---|
| [Rule](rule.md) applies to definitions or their relationships | The Rule states what must hold and identifies its scope. | Title must contain a non-whitespace character; Task Logic must be independent of the other project Components. |
| [Condition](condition.md) describes a relevant state | Its truth can determine applicability, readiness or the next action. | The supplied title contains a non-whitespace character. |
| [Check](check.md) verifies an expectation | It describes the method and what counts as success for a Rule, Condition or expected outcome. | Title validation checks valid and whitespace-only input against the Title Rule. |

A Rule expresses an obligation. A Condition expresses a statement that can be true or false. A Check describes how to examine an expectation. A Rule can include a Condition that determines when the obligation applies: when a change affects a UseCase, its guide must be updated.

A Condition can also control a Workflow. If a Check fails, the change returns for revision. The project can require a separate Rule that Checks must pass before integration.

The connections matter: the Check identifies the particular Rule or outcome it examines, and the Rule identifies the Concepts, Components, Artifacts or activities it governs. Stating a Check defines a planned verification method. A recorded execution supplies evidence for a particular run and its inputs.

## Work and concrete situations

| Connection | Meaning | Tasks example |
|---|---|---|
| [Workflow](workflow.md) orders activities and uses [Procedures](procedure.md) | The Workflow coordinates work; a Procedure explains how an activity is carried out. | Deliver a change uses Prepare a change, Verify a change and Review a change. |
| Actor participates in a Workflow or carries out a Procedure | The role has a stated responsibility in the work. | Contributor prepares and verifies a change; Reviewer reviews it. |
| Procedure refers to relevant definitions and Checks | Its steps identify what to work on and how to establish the expected result. | Verify a change selects affected Rules and Scenarios, performs their Checks and records the results. |
| Condition controls a Workflow path | The statement determines readiness or which activity follows. | Review begins when the selected Checks have passed. |
| [Scenario](scenario.md) illustrates a UseCase or Workflow | A concrete starting situation and actions lead to a stated expected outcome. | Adding Buy milk to an empty list creates an open Task. |
| [Decision](decision.md) explains a choice affecting definitions | The Decision names the choice, its reasons and the parts of the description it affects. | Local JSON storage explains the Local Storage Component and its associated Artifacts. |

A UseCase describes an action offered by the software and its outcome. A Procedure describes how someone carries out an activity. It may include using a UseCase or performing a Check. For example, Add task describes application behavior, while Review a change describes review steps.

A Workflow describes a repeatable arrangement of activities and participants. A Scenario illustrates one particular case. One Procedure can be reused by several Workflows, and one UseCase can have several Scenarios, including failure cases.

## Applying the relationships

Use the connection that expresses the intended meaning: an Actor uses an Interface; a Component provides it; an Artifact implements or documents it. The [Tasks example](task-management-example.md) follows these connections from a user's action through software structure, files, Rules, Checks and project work.

Return to the [terminology map](README.md).
