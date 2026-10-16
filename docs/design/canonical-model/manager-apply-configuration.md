# Configuration assignments and Manager mandates

Date: 16 October 2026. Proposed P01 configuration contract for the first reliable Apply, incorporating the [working-day discussion](../../concepts/reflections/desired-working-day-20261016.md). The [journal contract](manager-apply-journal.md) owns recorded transitions and storage. This design extends the [ports](manager-apply-port-contracts.md) and [rebuild plan](manager-apply-rebuild-plan.md); the formats below are proposed, not supported by the current CLI.

## Give each source one responsibility

| Source | Maintained content | Library example |
| --- | --- | --- |
| Canonical model | Intended behavior, required outputs, architecture and binding interfaces. | `loan-policy.spec.maxOpenLoans` is five in M1 and three in M2; existing loans remain returnable. |
| `.markitect/execution.yaml` | Manager profiles/instructions, creation areas, finite execution ceilings, Package overrides, check selection and notification preferences. | Lending uses the selected Manager profile and at most three review rounds. |
| Explicit host profile file | Literal executables, pinned protocol/tool identities, model/effort, controlled environment and instruction inputs. Supplied by the operator; no implicit profile discovery. | The selected Codex build and local test executable. Credentials are resolved locally and are not copied into model or journal payloads. |
| Assignment records | The owner's goal, constraints, granted execution scope, remaining limits and any explicit amendments. | Realize the library; later continue a specifically authorized limit change. |
| Registry | Versioned actual artifact ownership and Definition links. | Lending owns a particular source file; a parent cannot acquire it through a prompt. |
| Host operation contract | Artifact-only Apply, required independent review, source/candidate bindings and guarded publication. | Execution configuration cannot disable review or authorize model edits. |
| Mandate view | Computed, source-bound explanation of the above for one Manager/task/role. | Lending can choose its implementation within the architecture and coordinate through its parent. |

The model remains language v1. None of these runtime records introduces a canonical Assignment, Manager, Permission or Question Kind. Required behavior is not copied into execution settings. In particular, the loan limit has no configuration override.

## Project execution file

Use one versioned YAML file outside Definition roots. This illustrative configuration assumes P02 supplies the named profiles and selects a Go evaluation contract; it does not select that interface for M1 by itself. Profile references must resolve before the product can execute it.

```yaml
apiVersion: markitect.execution/v1
strategy: recursive-apply-v1
limits:
  maxRuns: 3
  maxSessionStarts: 80
  maxSteps: 160
  assignmentTimeout: 2h
defaults:
  profiles:
    manager: library-agent
    reviewer: library-agent
    verifier: library-agent
  reviewRounds: 3
  stepTimeout: 10m
  creationRoots: [src, tests, docs]
packages:
  lending:
    stepTimeout: 8m
checks:
  - id: project-tests
    profile: library-tests
    args: [test, ./...]
    timeout: 2m
notifications:
  questions: grouped
  incidents: immediate
```

The numeric limits are example ceilings, not permission to launch Agents in this design session. `creationRoots` are repository-relative directories for new artifact claims, not a required file layout. P02 must account for any root build files and protected assessment inputs explicitly. Existing artifacts still require current registry ownership. A directory permission does not grant another Manager's files or protected model/configuration paths.

`limits` applies across the assignment's runs. Count run creation, session starts and steps separately: a persistent session can execute several turns. Unknown dispatch remains charged. `assignmentTimeout` becomes an absolute recorded deadline when the assignment starts; waiting and successor runs do not reset it. Each step/check also has a finite timeout. Budgets control declared execution, not a guaranteed provider invoice.

`defaults` applies to every local Package Manager, including the ordinary root. `packages` keys are exact local Package paths; `.` identifies the root in this configuration syntax. An unknown or imported Package key is an error, not a request to create a Manager. Integration review uses the reviewer profile in a fresh session; final verification uses the verifier profile. The first strategy fixes Manager-plus-reviewer and sequential writing; other teams remain later supported strategies.

Custom instructions are optional file lists under `defaults.instructions` or `packages.<path>.instructions`, keyed by `manager`, `reviewer` or `verifier`. Files are explicit inputs captured with their original bytes. A task's outcome prompt is still supplied by its parent; the user's initial instruction is supplied to root only. Common custom instructions do not independently commission every Manager.

## Resolve settings deterministically

Resolve defaults, then Package overrides along the root-to-current-Package path. Profile references replace the corresponding role value. Instruction lists append in declared order, from broad to local. The remaining supported overrides may narrow `stepTimeout`, `reviewRounds` and `creationRoots`; an attempted expansion of an inherited ceiling is an error. New creation directories must be equal to or descendants of an allowed directory. There is no generic deep merge, wildcard Package override or hidden environment interpolation.

The effective grant is bounded by the project ceilings, the assignment's explicit grant and any parent-task restrictions. Shared assignment counters are not copied into independent per-Manager allowances. Review rounds count submitted candidates for each local/integration task; a successor receives fresh assessment tasks but consumes the existing assignment's run/start/step allowance.

Store effective values with the file/content/field origins that supplied them. Reject duplicate or unknown keys, unresolved profiles/instruction files, invalid Package paths, unbounded/nonpositive limits and unsupported strategy/role values before dispatch. Use bounded restricted YAML without aliases, custom tags or implicit null values. Durations are explicit positive duration strings. Configuration formats have their own schema; they are not canonical-model Definitions.

Structural checks reject forbidden options such as disabling mandatory review. Prose contradictions between custom instructions and model obligations remain questions for grounded assessment. No precedence rule lets instructions, configuration or a parent prompt weaken a binding model requirement.

## Host profiles and operational state

The operator supplies an explicit profile file and an absolute local state directory outside the governed repository. Their selection is recorded in acquisition. The profile file defines `agentProfiles` and `checkProfiles` with typed versioned adapter-specific sections. The project refers to their names; `run` receives prepared profile identities, not provider wire objects or a general property bag.

| Profile | Required preparation |
| --- | --- |
| Agent | Adapter kind, absolute executable path, exact executable/protocol-schema identity, model and effort, finite transport limits, effective provider settings/tool schemas and explicit environment/instruction policy. |
| Check | Absolute executable identity, fixed environment policy, argument prefix and finite output/termination limits. Project arguments are appended literally; no shell command string is interpreted. |
| State location | Repository/store identity, confined private directory and local SQLite journal configuration. Validate its identity before resuming an assignment. It is operational state, not generated project output. |

Selected actual bytes and effective identities enter the input binding. Recheck mutable executable/configuration dependencies at effect boundaries and report drift. Do not persist secrets merely to hash or reproduce an environment. Capture safe configuration and instruction bytes, secret references and identity bindings; explicitly report the observation boundary. A credential rotation is not an implicit business-rule change, but required identity/preparation checks still apply.

The [adapter plan](manager-apply-rebuild-plan.md#bind-the-executable-before-real-work) owns the first Codex protocol capability gate. This document does not choose a model, accept an installed alpha build or promise replay of the provider's internal prompt.

## Assignment run task and attempt

| Identity | Meaning and lifetime |
| --- | --- |
| Assignment | Stable owner goal and execution grant. It links successive runs, questions, consumed effort and amendments. It owns no second scheduler. |
| Apply run | One immutable model/configuration/source binding and its reconciliation. A changed acquired basis or effective execution policy requires a successor run; normal changes inside its candidate do not. |
| Task | One Manager responsibility under that run. New-run tasks have new IDs with explicit predecessor links where work is carried forward. |
| Attempt | Bounded rework of a task against that run's fixed basis. It does not reset assignment budgets. |
| Session and step | Actual role conversation and one dispatch, independently identified and recorded. They never supply assignment authority. |

The initial assignment stores owner input, goal, constraints, authorized operation target, project identity, explicit ceilings and deadline, plus the source of that authority. Omitted grant limits use the selected project ceilings. Renewing an exhausted limit requires an explicit recorded amendment; starting another run grants nothing by itself. The initial product has one local operator as decision owner; it does not invent an identity/authorization service from Agent-reported names.

An amendment cannot bypass project ceilings. If those need expansion, an authorized configuration edit and a new bound run are required as well. Changes to effective execution grants/deadlines are recorded in the successor basis, preserving consumption. An answer that only supplies existing-rule context can continue the same run.

The primary M1-to-A1 and M2-to-A2 experiment can be one assignment explicitly commissioned to perform both stages. Two unrelated owner requests can instead be separate assignments. A completed assignment is not silently reopened for a new objective. An accepted decision can authorize a named successor within the existing goal/grant; it does not authorize merge, release, deployment or production-data edits.

## Compile the effective mandate

`organization` renders an inspectable `MandateView` from supplied facts. `run` supplies task/assignment facts through plain organization inputs, avoiding an organization-to-run import. The view carries its content identity and original source references; `run` stores and binds it to the current input/task record. It is supplied to the Manager and its reviewer with their appropriate role capabilities.

The reviewer sees the Manager's mandate as assessment evidence and receives its own read-only grant. Supplying that mandate never grants the reviewer the Manager's write authority.

For Lending it explains:

- Responsibility: reconcile all local lending Definitions and coordinate reminders through its child.
- Binding basis: current policy, member eligibility, copy availability, returns and required outputs, with original passages available.
- Freedom: choose internal implementation within model architecture and interfaces; no additional business restrictions.
- Artifact authority: current owned files plus permitted new claims through Host; cross-owner changes return to their owner.
- Reserved choices: changed business rules, authority or protected inputs require the appropriate decision outside artifact Apply.
- Limits: remaining assignment allowance, local review/step limits and mandatory operation gates.

The view may display the bound loan limit, but refers to `loan-policy` as its authority. M2 changes the canonical value; no second mandate value needs editing. The reviewer examines material decisions and their source-bound authority as well as behavior. The Host validates typed boundaries; semantic authority judgments remain reviewable Agent work.

## Questions and owner attention

A Manager first retrieves existing knowledge, then decides within implementation freedom or coordinates through its common parent. It escalates a necessary decision beyond that mandate. The root has broader coordination scope, not unrestricted rights over business meaning.

Question records contain decision, source-grounded reason, recommendation/alternatives, exact proposed model edit when needed, and affected waiting/continuing work. Notification settings control grouping and timing only. A blocked assignment or significant execution uncertainty needs timely visibility; routine findings and local decisions remain inspectable without requiring an owner answer. The first UI can be CLI status and question inspection; no push service is required.

Actual owner answers enter through the application boundary and are bound to the question/proposal version and source basis. An Agent cannot answer as the owner by filling a JSON field. Accepting the exact displayed edit within the grant needs no second identical confirmation. Changed proposals, deferral and stale bases have the distinct transitions in the [journal contract](manager-apply-journal.md#questions-answers-and-model-changes).

## Remaining boundaries

Experience hints and a future operational doctor remain separate extensions. Explicit optional context can later include selected experience revisions; this does not require provider-native memory. Operational diagnosis has its own evidence/target and does not silently repair artifacts or promote experience into model rules.

P02 still chooses the exact library interface and independent executable expectations. Implementing these configuration contracts requires negative cases for inherited limits, unknown Packages/profiles, protected inputs, conflicting instructions and budget continuity. Format examples and design review are not evidence of a functioning product parser or real Agent run.

Return to the [journal](manager-apply-journal.md), [ports](manager-apply-port-contracts.md) or [Apply plan](manager-apply-implementation-plan.md).
