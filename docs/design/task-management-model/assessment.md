# Boundary and completeness review

This is an editorial pressure test of a proposed documentation model. It is not owner acceptance, an application test result or empirical evidence that Markitect prevents drift. The [original description](source-description.md) is the input; record bodies are the candidate canonical owners; [Relationships](relations.md) owns the graph.

## Content coverage

This table maps source topics to their definition owner. It is a review index, not a second normative definition.

| Original content | Canonical owner |
|---|---|
| Purpose: overview of open tasks | REQ-OVERVIEW |
| Local use on one machine | REQ-LOCAL |
| Command-line interaction | COMP-CLI |
| Tasks survive restart | REQ-DURABILITY |
| Add, list, complete | UC-ADD, UC-LIST, UC-COMPLETE; REQ-SCOPE owns offered scope |
| No shared use, sync or reminders | REQ-SCOPE |
| Task text and states | CONCEPT-TASK |
| Input text and initial open state | UC-ADD |
| Empty/whitespace text | CONCEPT-TASK; UC-ADD owns refusal outcome |
| Creation confirmation | UC-ADD; REQ-ERRORS owns common outcome protocol |
| List all texts/states; no-task result | UC-LIST |
| Unique selection, including identical wording | CONCEPT-TASK; UC-COMPLETE uses the reference |
| Complete open task and repeat completion | UC-COMPLETE |
| Unknown task failure | UC-COMPLETE |
| Interaction/task/storage responsibilities | COMP-CLI, COMP-MANAGEMENT, COMP-STORAGE |
| Concrete classes/program units remain open | Component definitions and DEC-LANGUAGE |
| Source, tests, user guide, agent hints | Artifact definitions |
| Guide: setup, start, actions, file location | ART-USER-GUIDE |
| Agent hints route to spec and development | ART-AGENT-GUIDE |
| Read/write error, success only after storage | REQ-ERRORS |
| Protect unreadable data | REQ-SAFE-STORAGE |
| Changes honor behavior and tests | REQ-CHANGE |
| Update changed requirements | REQ-CHANGE |
| Keep affected instructions current | REQ-DOCS |
| Action/failure/restart verification | Check definitions |
| Small scope and understandable realization | REQ-SCOPE and DEC-BOUNDARY |
| Local storage without a service | REQ-LOCAL and DEC-FILE |
| Open language/commands/file format | DEC-LANGUAGE, DEC-CLI, DEC-ENCODING |
| Technical choices must honor behavior | DEC-LANGUAGE and DEC-ENCODING within scoped Requirements |

## Editorial refinements, not inherited facts

The original description did not settle command syntax, English messages, exit codes, reference allocation, list order, text normalization, file selection, initial-file behavior or concurrency/crash guarantees. This draft selects concrete observable behavior in the appropriate record and explains the choices through Decisions. These selections remain proposed.

Language and storage representation remain deliberate implementation freedom. They must be concretized in the realized repository's tooling/storage documentation; they do not leave a domain action undefined. No task application files, CLI execution, implemented test suite or human benefit observation are claimed.

## Requirement versus UseCase

| Question | Chosen owner | Reason |
|---|---|---|
| What is a task and which states exist? | Concept | Shared domain meaning used by several actions. |
| What is the initial state of a new task? | UC-ADD | Local postcondition of that action. |
| What happens on repeated completion? | UC-COMPLETE | Local branch and caller-visible outcome. |
| Must completed tasks be shown, and in which order? | UC-LIST | Contract of that read action. |
| Must successful changes survive restart? | Requirement | Applies across mutation/read/process boundaries. |
| Must errors avoid destructive storage behavior? | Requirement | Shared obligation across every action. |
| How is a local postcondition checked? | Check verifies UseCase directly | No duplicate Requirement is introduced merely for traceability. |
| How is intent changed and documentation updated? | Requirements plus developer Artifact | A small project procedure fits without a new Workflow kind. |

The boundary works for this example as a modeling convention, not as a universal theorem. If one local rule becomes shared, give it a Requirement owner and replace the local authoritative clause with a reference in the same revision. Preserve its meaning and Check targets. A Concept's invariant can be verified directly; it need not be retyped as a Requirement.

## Pressure cases

| Case | Definition owner | Planned evidence |
|---|---|---|
| Add empty/whitespace/multiline text | CONCEPT-TASK, UC-ADD | CHECK-DOMAIN, CHECK-CLI |
| Two tasks with identical text | CONCEPT-TASK, UC-ADD | CHECK-ACTIONS, CHECK-RESTART |
| Empty/missing task file | REQ-SAFE-STORAGE, UC-LIST | CHECK-RESTART, CHECK-CLI |
| Complete unknown/already completed task | UC-COMPLETE | CHECK-ACTIONS, CHECK-CLI |
| Complete then restart | REQ-DURABILITY | CHECK-RESTART |
| Corrupt data/read failure/write failure | REQ-SAFE-STORAGE, REQ-ERRORS | CHECK-STORAGE-ERRORS |
| Default versus explicit file, changed working directory | COMP-CLI | CHECK-CLI |
| Conflicting writes or power loss | REQ-LOCAL declares scope boundary | Not claimed as supported or verified |
| Code correct but guide outdated | ART-USER-GUIDE, REQ-DOCS | CHECK-DOCUMENTATION |
| Tests pass but person struggles to find outstanding work | REQ-OVERVIEW | CHECK-OVERVIEW; machine evidence is insufficient |

## What the modeling exercise exposes

The seven kinds describe the example without adding a Project, Goal, Workflow, agent or evidence kind. Two precise relationship additions were needed beyond the first conversational sketch: `provided-by` for UseCase responsibility and `owned-by` for a planned Artifact's accountable component. `verifies` must permit direct UseCase/Concept targets as well as Requirements, otherwise local behavior gets duplicated merely to attach a test.

An Artifact is not the same thing as the Markdown document that defines it. A Component's technical responsibility is not organizational authority. A Decision's rationale is not its subject's executable obligation. Planned verification, executed evidence and human acceptance stay separate.

### Remaining Markitect design questions

- Should goals remain Requirements with an explicit strength, or need their own semantics later?
- Is Component the right general term when larger examples introduce systems/modules/products?
- Does a richer ordered development process eventually justify a Workflow kind?
- What structural/compiler effects should each relation have for context, impact and invalidation?
- How should artifact scopes become exact realized file inventories while preserving one owner?

These questions do not block reading or implementing this fictional application's v1 contract. They are unresolved product-language questions, not implemented Markitect features.
