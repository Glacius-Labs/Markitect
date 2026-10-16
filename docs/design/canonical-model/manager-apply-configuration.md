# Configuration assignments and Manager mandates

This contract owns the execution input fields and their deterministic resolution. The [execution contract](manager-apply-execution.md) owns budget admission, run transitions and status meaning; the [journal](manager-apply-journal.md) owns their durable records.

## Give each source one responsibility

| Source | Maintained content | Library example |
| --- | --- | --- |
| Canonical model | Intended behavior, required outputs, architecture and binding interfaces. | `loan-policy.spec.maxOpenLoans` is five in M1 and three in M2; existing loans remain returnable. |
| `.markitect/execution.yaml` | Manager profiles and instructions, creation areas, finite ceilings, Package overrides, check selection and notification preferences. | Lending uses the selected Manager profile and at most three review rounds. |
| Explicit host profile file | Literal executables, pinned protocol and tool identities, model and effort, controlled environment and instruction inputs. Supplied by the operator; no implicit discovery. | The selected Codex build and the local test executable. Credentials are resolved locally and never copied into model or journal payloads. |
| Assignment record | The owner's goal, constraints, granted caps and recorded extensions. | Apply the current library model. |
| Registry | Versioned artifact ownership and Definition links. | Lending owns a particular source file; a parent cannot acquire it through a prompt. |
| Host operation contract | Artifact-only Apply, required independent review, source and candidate bindings, guarded publication. | Configuration cannot disable review or authorize model edits. |
| Mandate view | A computed, source-bound explanation of the above for one Manager, task and role. | Lending may choose its implementation within the architecture and coordinates through its parent. |

The model stays language v1. None of these runtime records adds a canonical Assignment, Manager, Permission or Question Kind. Required behavior is never copied into execution settings; the loan limit in particular has no configuration override.

## Project execution file

One versioned YAML file outside the Definition roots. The figures below are illustrative, not measured sufficient capacity.

```yaml
apiVersion: markitect.execution/v1
strategy: recursive-apply-v1
limits:
  maxRuns: 5
  maxSessionStarts: 160
  maxSteps: 320
  maxExecutionTime: 16h
  maxStepTimeout: 10m
  hostOperationTimeout: 2m
  maxReviewRounds: 3
  maxContextBytes: 400000
  maxSessionSteps: 40
  maxTransientRetries: 2
  creationRoots: [realization]
assignment:
  runs: 3
  sessionStarts: 80
  steps: 160
  executionTime: 8h
completionReserve:
  sessionStarts: 24
  steps: 40
  executionTime: 2h
defaults:
  profiles:
    manager: library-agent
    reviewer: library-agent
  reviewRounds: 3
  batchNoChangeReviews: false
  stepTimeout: 10m
  creationRoots: [realization]
packages:
  lending:
    stepTimeout: 8m
checks:
  - id: project-tests
    profile: library-tests
    directory: realization
    args: [test, ./...]
    timeout: 2m
acceptance:
  ownerApproval: []
notifications:
  questions: grouped
  incidents: immediate
```

Everything under `limits` is a hard project ceiling. `assignment` sets the default caps of a new assignment; an owner extension can raise them up to `maxRuns`, `maxSessionStarts`, `maxSteps` and `maxExecutionTime`. `maxStepTimeout`, `maxReviewRounds` and `creationRoots` are ceilings, not consumable counters. `hostOperationTimeout` bounds each Host control operation outside the budget. `completionReserve` is part of the caps, not extra allowance; a reserve that the execution contract's preparation check rejects stops the run before dispatch.

`maxContextBytes` bounds a rendered context package; exceeding it stops preparation with a finding to split the Package. `maxSessionSteps` bounds one Manager session before a replacement session starts from the journal summary. `maxTransientRetries` bounds automatic retries of calls that provably sent nothing. `batchNoChangeReviews` lets one reviewer session assess several no-change deliveries, each with its own verdict. `acceptance.ownerApproval` lists Packages whose changes wait for the owner's approval before publication; it is empty by default, because the owner's commit already accepts the result.

`scope` optionally names the Kinds and reference Properties through which the model states a Definition's scope, for example `scope: [{kind: software/Rule, property: software/appliesTo}]`. A Definition whose scope covers a Package enters the context of that Package's Managers. The configuration never lists individual Definitions; M1 needs no `scope`, because its shared rules reach every area through explicit links.

`limits.creationRoots` bounds the permitted creation areas; defaults and Package settings select within it. Here the code, tests and documentation live below `realization/`. Existing artifacts still need registry ownership, and a directory permission grants no other Manager's files or protected model and configuration paths. A check's `directory` is confined to its captured candidate, and its arguments are literal, never a shell command.

`defaults` applies to every local Package Manager, including the root. `packages` keys are exact local Package paths, and `.` is the root. An unknown or imported Package key is an error, not a request to create a Manager. Local and integration reviews use the reviewer profile in fresh sessions. Once a second session adapter exists, the default reviewer profile uses a different model family than the Manager profile ([ASM-005](../../concepts/register.md#assumptions-and-hypotheses)); the final evidence check is deterministic Host work. The first strategy is Manager plus reviewer with sequential writing.

Custom instructions are optional file lists under `defaults.instructions` or `packages.<path>.instructions`, keyed by `manager` or `reviewer`, captured with their original bytes. Instructions carry operational guidance only, such as reporting style or tool usage; a project rule in them is a review finding and belongs in the model ([concept](manager-apply-concept.md#four-kinds-of-input-and-record)). The parent still supplies each task's outcome prompt, and the user's instruction goes to the root only.

## Resolve settings deterministically

Resolve project defaults, then the exact Manager override only; there is no inheritance from ancestor overrides. Profile references replace the role's value. Instructions append the project defaults, then the exact override, in declared order. `stepTimeout`, `reviewRounds` and `creationRoots` replace the default within the project ceilings; a default is not a ceiling, so a default of two review rounds can be overridden to three where `maxReviewRounds` allows it. Creation directories must equal or descend from an allowed root. There is no deep merge, wildcard override or environment interpolation.

In the example, lending gets eight minutes per step while lending/reminders keeps the ten-minute default, and a `.` override configures the root only. Restrictions in a delegated task are a separate runtime narrowing that still constrains the child. Effective values are stored with the file and field that supplied them, so every bound is explainable.

Reject duplicate or unknown keys, unresolved profiles or instruction files, invalid Package paths, unbounded or nonpositive limits and unsupported strategy or role values before dispatch. Use restricted YAML without aliases, custom tags or implicit nulls; durations are explicit positive strings. Structural checks reject forbidden options such as disabling mandatory review. A prose conflict between custom instructions and a model obligation is a question for grounded assessment; no precedence rule lets instructions, configuration or a parent prompt weaken a binding requirement.

## Host profiles and operational state

The operator supplies an explicit profile file and an absolute state directory outside the governed repository; acquisition records both. The profile file defines `agentProfiles` and `checkProfiles` with typed, versioned, adapter-specific sections. The project refers to them by name, and `run` receives prepared profile identities, not provider wire objects.

| Profile | Required preparation |
| --- | --- |
| Agent | Adapter kind, absolute executable path, exact executable and protocol-schema identity, model and effort, finite transport limits, effective provider settings and tool schemas, explicit environment and instruction policy. |
| Check | Absolute executable identity, fixed environment policy, argument prefix and finite output and termination limits. Project arguments are appended literally. |
| State location | Repository and store identity, a confined private directory and the SQLite journal configuration. Its identity is validated before an assignment resumes. |

The selected bytes and identities enter the run's input binding and are rechecked at effect boundaries. Secrets are never persisted; capture safe configuration and instruction bytes, secret references and identity bindings, and report the observation boundary. The [rebuild plan](manager-apply-rebuild-plan.md#bind-the-executable-before-real-work) owns the Codex capability gate.

## Assignment run task and attempt

| Identity | Meaning and lifetime |
| --- | --- |
| Assignment | One owner request with its caps. It links its runs, questions, consumption and extensions and owns no scheduler. |
| Apply run | One immutable model, configuration and source binding and its reconciliation. A changed basis or execution policy needs a continuation run after this one finished. |
| Task | One Manager responsibility within a run. Tasks in a continuation get new IDs with predecessor links where work carries forward. |
| Attempt | One submission within a task against the run's fixed basis. It resets no budget or round counter. |
| Session and step | An actual role conversation and one dispatch. They never supply authority. |

The assignment stores the owner input and goal, constraints, operation target, project identity, caps and authority source. Omitted caps use the `assignment` defaults. Renewing an exhausted cap needs a recorded extension within the project ceilings; raising a ceiling needs an authorized configuration edit. Starting another run grants nothing. The first product has one local operator as decision owner and no identity service based on Agent-reported names. An accepted decision can authorize a continuation run within the assignment; it never authorizes merge, release, deployment or production-data edits.

## The mandate section

`organization` renders the mandate as a section of the context package, not as a separate record; `run` passes task and assignment facts as plain inputs, so organization does not import run. The section carries its source references and is bound to the task record with the rest of the package. The Manager and its reviewer both receive it; the reviewer uses it as evidence under its own read-only grant, never as write authority.

For lending it states:

- Responsibility: reconcile all local lending Definitions and coordinate reminders through its child.
- Binding basis: the current policy, member eligibility, copy availability, returns and required outputs, with the original passages available.
- Freedom: the internal implementation within the model's architecture and interfaces; no additional business restrictions.
- Artifact authority: its owned files plus new paths claimed through the Host; changes to other owners' files go to those owners.
- Reserved choices: changed business rules, authority or protected inputs need a decision outside artifact Apply.
- Limits: the remaining assignment capacity, local step and round limits and the mandatory gates.

The section may display the bound loan limit but cites `loan-policy` as its authority, so M2 changes one canonical value and no mandate text. The reviewer checks material decisions against their source-bound authority as well as their behavior. The Host validates the typed boundaries; judging semantic authority remains reviewable Agent work.

## Questions and owner attention

A Manager first retrieves existing knowledge, then decides within its freedom or coordinates through its common parent. It escalates only a necessary decision beyond its mandate. The root has a broader coordination scope, not more authority over business meaning.

A question records the decision needed, its source-grounded reason, a recommendation and alternatives, the exact proposed model edit when one is needed, and the waiting and continuing work. Notification settings control only grouping and timing. A blocked assignment or a serious execution uncertainty needs timely visibility; routine findings and local decisions stay inspectable without an answer. CLI status and question inspection suffice for the first version.

Owner answers enter through the application boundary, bound to the question version and its source basis. An Agent cannot answer as the owner by filling a field. Accepting the exact displayed edit needs no second confirmation. Changed proposals, deferral and stale bases follow the [journal contract](manager-apply-journal.md#questions-answers-and-model-changes).

Configuration needs negative cases for exact overrides versus task restrictions, ceilings, unknown Packages and profiles, invalid reserves, protected inputs, conflicting instructions, oversized contexts and unknown approval Packages. Experience hints and an operational doctor are later extensions ([north star](manager-realization-north-star.md#later-operations-and-extensions)).
