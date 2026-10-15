# Tasks: a relationship example

Status: illustrative project specification for discussing the [terms](README.md) and their [relationships](relationships.md). The paths below name intended files in the example Tasks repository. The behaviors and Checks describe the expected application and its verification.

## Project, actors and interface

The **Project**, Tasks, helps a person keep a local task list. A person can add tasks, list them and mark them complete. Tasks remain available after the application exits.

| Actor | Context and responsibility |
|---|---|
| User | Uses the application to manage tasks. |
| Contributor | Changes project files and carries out verification. |
| Reviewer | Examines a proposed change and its verification evidence. |

The **Interface**, Tasks command line, exposes Add task, List tasks and Complete task. The Command Line Component provides this Interface. User reaches the UseCases through it. The same person can act as User and Contributor at different times.

## Concepts, rules and conditions

| Concept | Meaning and relationships |
|---|---|
| Task | An item of work with an Identifier, Title and TaskStatus. |
| Identifier | A positive integer that uniquely identifies a Task in the stored list. |
| Title | The text describing the work. |
| TaskStatus | The Task's state: open or completed. |

These **Rules** apply to the Concepts and UseCases:

| Rule | Scope and obligation |
|---|---|
| Meaningful title | Title must contain at least one non-whitespace character. Add task removes surrounding whitespace before validation and storage. |
| Stable identity | Each Task has a unique Identifier. A new Task receives the next integer above the highest existing Identifier, starting at 1 for an empty list. Completion preserves it. |
| Task lifecycle | Add task creates an open Task. Complete task changes an open Task to completed. Completing a completed Task succeeds and preserves its state. |
| Persistent change | A successful Add task or Complete task leaves its result in local storage. Success is reported after the save succeeds. |
| Rejected operation | Invalid input or an unknown Identifier produces a clear error and preserves stored Tasks. |

The following **Conditions** name statements used in these operations:

| Condition | Role in the example |
|---|---|
| Title is meaningful | After trimming, the supplied title contains a non-whitespace character. Determines whether Add task can proceed. |
| Task exists | A stored Task has the supplied Identifier. Determines whether Complete task can proceed. |
| Task is completed | The selected Task has completed status. Selects the already-completed path of Complete task. |

The Meaningful title Rule requires an acceptable title. The Title is meaningful Condition describes whether particular input meets that requirement. The Title validation Check below describes how the requirement is verified.

## UseCases and their scenarios

Application handles these **UseCases**, reached through the command-line Interface:

| UseCase | Action and expected outcome |
|---|---|
| Add task | Accept a title. When Title is meaningful, assign an Identifier, create an open Task and save it. Report the Identifier, stored Title and open status after saving. Whitespace-only input produces an error and preserves the list. |
| List tasks | Read and display every stored Task with its Identifier, Title and TaskStatus, ordered by Identifier. An empty list displays that there are no Tasks. |
| Complete task | Accept an Identifier. When Task exists, ensure that its status is completed and save any change. Report completed status. An already-completed Task succeeds with the same Identifier and state. An unknown Identifier produces an error and preserves the list. |

Input that cannot be read as the positive integer required for Complete task produces a clear error and preserves the list. A missing storage file represents an empty list. Unreadable or invalid stored data produces an error. A failed save produces an error; the previous valid stored data is preserved. Add task and Complete task make their results visible to a later List tasks operation.

Each **Scenario** supplies concrete inputs and an expected result:

| Scenario | Starting situation and action | Expected outcome |
|---|---|---|
| Add the first task | The list is empty. User adds `  Buy milk  `. | Task 1 is saved with Title `Buy milk` and open status. |
| Reject an empty title | Task 1 exists. User adds a title consisting of spaces. | A title error is reported; Task 1 is preserved and no new Task is stored. |
| Complete a task | Task 1 is open. User completes Identifier 1. | Task 1 is saved with completed status and its existing Title. |
| Complete a completed task | Task 1 is completed. User completes Identifier 1 again. | The operation succeeds; Task 1 retains its Identifier, Title and completed status. |
| Reject an unknown task | Only Task 1 exists. User completes Identifier 9. | An unknown-task error is reported; Task 1 is preserved. |
| Read after restart | Task 1 has been completed and saved. User exits, starts a new application invocation and lists tasks. | Task 1 appears once with its Title and completed status. |
| Read an empty list | Storage is absent. User lists tasks. | The Interface reports that there are no Tasks. |
| Handle invalid stored data | The storage file contains invalid JSON. User lists tasks. | A storage error is reported and the file is preserved. |
| Handle a failed save | Task 1 exists. User adds another Task while storage cannot accept a write. | A storage error is reported; success is withheld and the previously saved list is preserved. |

## Components and artifacts

The Tasks application contains four **Components**:

| Component | Purpose and responsibility |
|---|---|
| Task Logic | Defines the Concepts and their task-specific behavior, including title validation and lifecycle rules. |
| Application | Handles the UseCases and coordinates Task Logic with Local Storage. |
| Command Line | Provides the command-line Interface, translates input into UseCase requests and displays results. |
| Local Storage | Reads and saves Tasks in a local JSON file. |

Command Line uses Application. Application uses Task Logic and Local Storage. Local Storage uses Task Logic's task representation. These are source dependencies between Components. Task Logic is independent of the other project Components. This last statement is an architecture Rule, examined by the Dependency boundary Check.

Each **Artifact** description identifies intended repository files and their purpose:

| Artifact | Intended path | Relationship |
|---|---|---|
| Task Logic source | `src/task_logic/` | Implements the Concepts and task behavior. |
| Application source | `src/application/` | Implements the UseCases. |
| Command-line source | `src/cli/` | Implements the Interface and delegates requests to Application. |
| Storage source | `src/local_storage/` | Implements local JSON persistence. |
| Task Logic tests | `tests/task_logic/` | Verifies title, identity and lifecycle Rules. |
| Application tests | `tests/application/` | Verifies the UseCase Scenarios and storage-error outcomes. |
| Command-line tests | `tests/cli/` | Verifies that the Interface exposes the UseCases and reports their outcomes. |
| UseCase guide | `docs/use-cases.md` | Documents the UseCases, inputs, outcomes and Scenarios. |
| Architecture guide | `docs/architecture.md` | Documents Components, dependencies and the storage Decision. |
| Contribution guide | `CONTRIBUTING.md` | Documents the change Workflow, its Procedures and review Rules. |

A documentation Rule requires the UseCase guide to describe each UseCase and its successful and failure outcomes. When a change affects a UseCase, Contributor updates the corresponding description and Scenarios. The Documentation alignment Check examines that relationship.

## Checks and the expectations they verify

| Check | Expectations examined | Method and success criterion |
|---|---|---|
| Title validation | Meaningful title Rule; Add the first task and Reject an empty title Scenarios | Exercise padded meaningful text, empty text and whitespace-only text. Pass when meaningful text is trimmed and stored, invalid text is rejected and rejected operations preserve existing Tasks. |
| Task behavior | Stable identity and Task lifecycle Rules; completion Scenarios | Exercise multiple additions, completion and repeated completion. Pass when Identifiers are allocated as described, remain unique and stable, and statuses match the expected outcomes. |
| Persistence and errors | Persistent change and Rejected operation Rules; restart and storage-failure Scenarios | Use a temporary store across application invocations and arrange read/write failures. Pass when saved Tasks survive restart, errors are reported and failures preserve previously saved data. |
| Command-line behavior | Interface-to-UseCase relationships | Exercise each offered action through the command line. Pass when valid requests reach the corresponding UseCase, the documented outcomes are displayed and malformed Identifiers produce clear errors. |
| Dependency boundary | Task Logic architecture Rule | Inspect source imports in Task Logic. Pass when none imports Application, Command Line or Local Storage. |
| Documentation alignment | UseCase documentation Rule | Compare changed UseCases and Scenarios with the UseCase guide. Pass when each offered action, its inputs and its successful and failure outcomes agree with their descriptions. |

A Check specifies planned verification. Contributor records the candidate revision, selected Checks, observations and outcomes when performing verification. Reviewer uses that evidence to assess the change and its coverage.

## Workflow, procedures and conditions

The **Workflow**, Deliver a change, coordinates project work:

| Activity | Actor and Procedure | Next activity |
|---|---|---|
| Prepare | Contributor carries out Prepare a change. | Verify the candidate. |
| Verify | Contributor carries out Verify a change. | When Selected checks passed is true, proceed to Review. When false, return to Prepare. |
| Review | Reviewer carries out Review a change. | When Review accepted is true, the change is ready for integration. When false, return to Prepare and repeat verification and review. |

**Selected checks passed** means every selected Check has a recorded passing outcome for the current candidate revision. **Review accepted** means Reviewer has recorded acceptance of that same revision. A change to the candidate starts a new verification and review cycle.

A review Rule requires each change to complete verification and review before integration. These Workflow Conditions determine which activity follows. The Rule supplies the obligation to follow that path.

The reusable **Procedures** explain how to perform each activity:

| Procedure | Steps | Expected result |
|---|---|---|
| Prepare a change | Identify affected definitions and files. Update the behavior description, implementation, documentation and relevant Checks. Record the candidate revision. | A candidate with an explained change and identified verification scope. |
| Verify a change | Select Checks covering affected Rules, Scenarios, Components and Artifacts. Perform them against the candidate. Record methods, revision and results, including failures. | Verification evidence for the candidate and a truth value for Selected checks passed. |
| Review a change | Examine the changed descriptions and files, the selected verification scope and the recorded evidence. Record acceptance or required revisions. | A recorded review outcome for the candidate and a truth value for Review accepted. |

For a concrete **Workflow Scenario**, a Contributor changes title validation but forgets to update a corresponding Scenario in the guide. Documentation alignment fails, so Selected checks passed is false. The Workflow returns to Prepare. Contributor updates the guide, records a new candidate and repeats verification. When the selected Checks pass, Reviewer examines that candidate. Acceptance makes it ready for integration.

UseCases describe the software's task operations. These Procedures explain project activities. The Workflow orders those activities; the Workflow Scenario illustrates one particular passage through them.

## Decision and its consequences

The **Decision**, Local JSON storage, selects a local JSON file because Tasks serves one person's local list and benefits from straightforward inspection and backup.

This choice affects Local Storage, its source Artifact, the Architecture guide and the persistence Checks. The Persistent change and Rejected operation Rules continue to define the required behavior of that storage choice.

## What this example establishes

All fourteen terms have a concrete role and can be connected using ordinary language. The example distinguishes offered behavior, domain meaning, software structure, project files, obligations, verification and project activities. Actor now covers both application interaction and participation in project work, with each role's context stated explicitly.

The next refinement can use a particular relationship or example statement as its starting point, such as a Component's required Artifacts or a Check's coverage of a Rule.

Return to the [relationships](relationships.md) or the [terminology map](README.md).
