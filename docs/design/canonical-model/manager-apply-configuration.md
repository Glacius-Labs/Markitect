# Configuration assignments and Manager mandates

This contract owns the execution input fields and their deterministic resolution. The [execution contract](manager-apply-execution.md) owns budget admission, run transitions and status meaning; the [journal](manager-apply-journal.md) owns their durable records.

This is a target contract, not evidence that preparation or its input decoders are implemented. The strategy defaults below preserve accepted effects; the exact ignore-file format is a new preparation input choice.

## Give each source one responsibility

| Source | Maintained content | Library example |
| --- | --- | --- |
| Canonical model | Intended behavior, required outputs, architecture and binding interfaces. | `loan-policy.spec.maxOpenLoans` is five in M1 and three in M2; existing loans remain returnable. |
| `.markitect/execution.yaml` | Manager profiles and instructions, creation areas, finite ceilings, Package overrides, scope selection, check selection and owner approval. | Lending uses the selected Manager profile and at most three review rounds. |
| Product strategy defaults | Versioned defaults shipped with the product, with their original bytes and source fields. | Absent review-round configuration takes three from the strategy document. |
| `.markitect/ignore.yaml` | Optional exact-file accounting exceptions with reasons. | One observed library editor-state file is ignored. |
| Explicit host profile file | Literal executables, pinned protocol and tool identities, model and effort, controlled environment and instruction inputs. Supplied by the operator; no implicit discovery. | The selected Codex build and the local test executable. Credentials are resolved locally and never copied into model or journal payloads. |
| Assignment record | The owner's goal, constraints, granted caps and recorded extensions. | Apply the current library model. |
| Registry | Versioned artifact ownership and Definition links in `.markitect/registry/`. | Lending owns a particular source file; a parent cannot acquire it through a prompt. |
| Host operation contract | Artifact-only Apply, required independent review, source and candidate bindings, guarded publication. | Configuration cannot disable review or authorize model edits. |
| Mandate section | A computed, source-bound explanation of the above for one Manager, task and role, rendered into its context package. | Lending may choose its implementation within the architecture and coordinates through its parent. |

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
    requiredBy: library/callable-interface
    timeout: 2m
acceptance:
  ownerApproval: []
```

Everything under `limits` is a hard project ceiling. `assignment` sets the default caps of a new assignment; an owner extension can raise them up to `maxRuns`, `maxSessionStarts`, `maxSteps` and `maxExecutionTime`. `maxStepTimeout`, `maxReviewRounds` and `creationRoots` are ceilings, not consumable counters. `hostOperationTimeout` bounds each Host control operation outside the budget. `completionReserve` is part of the caps, not extra allowance; a reserve that the execution contract's preparation check rejects is a preparation failure, and no run is prepared.

`maxContextBytes` bounds a rendered context package ([concept](manager-apply-concept.md#context-packages)). `maxSessionSteps` and `maxTransientRetries` bound one Manager session and automatic retries ([execution](manager-apply-execution.md#admission-amendments-and-finite-correction)). `batchNoChangeReviews` enables batched no-change reviews ([concept](manager-apply-concept.md#distinct-review-responsibilities)). `acceptance.ownerApproval` lists Packages whose changes need the owner's approval before publication, asked as an ordinary question; it is empty by default, because the owner's commit already accepts the result.

`scope` optionally names the Kinds and Properties through which the model states a Definition's scope, for example `scope: [{kind: software/Rule, property: software/appliesTo}]`. The Property's values are local Package paths; `.` stands for the whole model, and the organization rejects unknown paths before dispatch. A Definition whose scope lists a Package or one of its ancestors enters the context of that Package's Managers. The configuration never lists individual Definitions; M1 needs no `scope`, because its shared rules reach every area through explicit links.

A check that executes a command the model requires names that Definition in `requiredBy`, and `check` verifies that the Definition exists. Prose cannot be executed, so the command itself is repeated literally and review keeps both in step; this is the one accepted repetition of model content in configuration ([DEC-033](../../concepts/register.md#dec-033-binding-intent-lives-only-in-the-model)).

`limits.creationRoots` bounds the permitted creation areas; defaults and Package settings select within it. Here the code, tests and documentation live below `realization/`. Existing artifacts still need registry ownership, and a directory permission grants no other Manager's files or protected model and configuration paths. A check's `directory` is confined to its captured candidate, and its arguments are literal, never a shell command.

`defaults` applies to every local Package Manager, including the root. `packages` keys are exact local Package paths, and `.` is the root. An unknown or imported Package key is an error, not a request to create a Manager. Local and integration reviews use the reviewer profile in fresh sessions; the final evidence check is deterministic Host work. Reviewer profiles should use a different model family than Manager profiles, for example Claude Code reviewing Codex ([ASM-005](../../concepts/register.md#assumptions-and-hypotheses)). The same family is allowed, and the run report states each role's profile; the example keeps one profile for brevity. The first strategy is Manager plus reviewer with sequential writing.

Custom instructions are optional file lists under `defaults.instructions` or `packages.<path>.instructions`, keyed by `manager` or `reviewer`. Each entry identifies an exact repository-relative regular file. Preserve list order, repeated entries and original captured bytes; do not expand directories or globs, normalize paths or deduplicate the list. An existing instruction file without captured bytes leaves preparation `NotCompleted` and requires a new complete capture; preparation cannot extend an old capture or its guard. Instructions carry operational guidance only, such as reporting style or tool usage; a project rule in them is a review finding and belongs in the model ([concept](manager-apply-concept.md#four-kinds-of-input-and-record)). The parent still supplies each task's outcome prompt, and the user's instruction goes to the root only.

## Strategy defaults and their sources

For `recursive-apply-v1`, these accepted effects apply when the corresponding project setting is absent:

| Effective setting | Default |
| --- | --- |
| `reviewRounds` | `3` |
| Manager and reviewer instruction lists | `[]` each |
| `batchNoChangeReviews` | `false` |
| `acceptance.ownerApproval` | `[]` |

These values must come from one real, versioned product-owned defaults document shipped with the product. Preparation retains its original bytes and digest, using `product:recursive-apply-v1/defaults.yaml` and the actual field pointer in that document as the source. A missing document leaves preparation `NotCompleted`; it cannot be replaced with invented provenance. This contract does not create that document or prescribe an implementation package.

`AuthoredExecution` retains absence. Resolution obtains an absent effective value from the captured strategy document; an explicitly configured value keeps its authored source. Effective defaults must satisfy the project ceilings: an absent review-round setting with `maxReviewRounds: 2` fails because the effective default is three. Never clamp the value or cite a nonexistent execution field as its source.

## Resolve settings deterministically

Resolve project defaults, then the exact Manager override only; there is no inheritance from ancestor overrides. Profile references replace the role's value. Instructions append the project defaults, then the exact override, in declared order. `stepTimeout`, `reviewRounds` and `creationRoots` replace the default within the project ceilings; a default is not a ceiling, so a default of two review rounds can be overridden to three where `maxReviewRounds` allows it. Creation directories must equal or descend from an allowed root. There is no deep merge, wildcard override or environment interpolation.

In the example, lending gets eight minutes per step while lending/reminders keeps the ten-minute default, and a `.` override configures the root only. Restrictions in a delegated task are a separate runtime narrowing that still constrains the child. Effective values are stored with the file and field that supplied them, so every bound is explainable.

Reject duplicate or unknown keys, unresolved profiles or instruction files, invalid Package paths, unbounded or nonpositive limits and unsupported strategy or role values before dispatch. Use restricted YAML without aliases, custom tags or implicit nulls; durations are explicit positive strings. Structural checks reject forbidden options such as disabling mandatory review. A prose conflict between custom instructions and a model obligation is a question for grounded assessment; no precedence rule lets instructions, configuration or a parent prompt weaken a binding requirement.

## Exact file ignore policy

The optional `.markitect/ignore.yaml` has exactly `apiVersion: markitect.ignore/v1` and a required `files` list. Each item has exactly two string fields, `path` and `reason`; `reason` must contain non-whitespace text and retains its original text. `files: []` is an explicit empty policy. Use the restricted YAML rules above; there is no legacy-format fallback.

For example, when the complete Census contains this library editor-state file:

```yaml
apiVersion: markitect.ignore/v1
files:
  - path: .editor/library-session.json
    reason: Local editor layout for the library.
```

Each path names one exact repository-relative regular file in that same Census. There is no glob or directory expansion, path normalization or inference from an acquisition boundary. Wildcard-looking text is never expanded and must itself match one exact observed regular-file path. Reject unknown or duplicate keys, duplicate paths, blank reasons, missing files, directory entries and non-relative or non-exact paths. For example, `.editor/` and `../editor-state.json` are invalid; listing `.editor/library-session.json` after its deletion also fails.

A complete Census proving that `.markitect/ignore.yaml` is absent supplies an empty policy. If the file exists in Census but its input bytes were not captured, retain the observation and return `NotCompleted`; do not assume an empty policy or fetch bytes into an old guard. Exact observed files and their reasons supply the ignore classifications to `AccountFiles`.

An ignore entry overlapping an input or registered artifact fails. Input protection covers model manifests, locks and declared trees, plus exact configuration, registry, instruction and control inputs, including reserved or future input paths; it does not protect every file beneath an acquisition boundary. Gitignore never supplies coverage. Ignoring a file cannot satisfy or waive required realization, independent review or the Definition census.

## Host profiles and operational state

The operator supplies an explicit profile file and an absolute state directory outside the governed repository; acquisition records both. The profile file defines `agentProfiles` and `checkProfiles` with typed, versioned, adapter-specific sections. The project refers to them by name, and `run` receives prepared profile identities, not provider wire objects.

Host instruction inputs outside the repository remain explicit locations in adapter-owned configuration. Host profiles keep their inline typed, versioned, adapter-specific sections. The concrete driver fields, versions, schema and readiness checks belong to the actual consumer and adapter deliverable.

| Profile | Required preparation |
| --- | --- |
| Agent | Adapter kind (Codex App Server or Claude Code), runtime image ID, the CLI version and protocol-schema identity the image records, model and effort, credential reference, container and transport limits, effective provider settings and tool schemas, explicit environment and instruction policy. |
| Check | Runtime image ID, executable inside the image, fixed environment policy, argument prefix, container, output and termination limits. Project arguments are appended literally. |
| State location | Repository and store identity, a confined private directory, the SQLite journal configuration, and the container engine executable with the runtime's volume prefix. Its identity is validated before an assignment resumes. |

The selected bytes and identities enter the run's input binding and are rechecked at effect boundaries. Secrets are never persisted; capture safe configuration and instruction bytes, secret references and identity bindings, and report the observation boundary. A credential reference names an environment variable or a private file on the Host; its value goes only to the provider process inside the container ([rebuild plan](manager-apply-rebuild-plan.md#container-runtime)). The [rebuild plan](manager-apply-rebuild-plan.md#bind-the-runtime-image-before-real-work) also owns the image and handshake gate.

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
- Artifact authority: its owned files plus new paths claimed through the Host; changes to other owners' files are requested through the common parent.
- Reserved choices: changed business rules, authority or protected inputs need a decision outside artifact Apply.
- Limits: the remaining assignment capacity, local step and round limits and the mandatory gates.

The section may display the bound loan limit but cites `loan-policy` as its authority, so M2 changes one canonical value and no mandate text. The reviewer checks material decisions against their source-bound authority as well as their behavior. The Host validates the typed boundaries; judging semantic authority remains reviewable Agent work.

## Questions and owner attention

A Manager first retrieves existing knowledge, then decides within its freedom or coordinates through its common parent. It escalates only a necessary decision beyond its mandate. The root has a broader coordination scope, not more authority over business meaning.

The [journal contract](manager-apply-journal.md#questions-answers-and-model-changes) owns the content of questions, how owner answers are bound and what they trigger. A blocked assignment or a serious execution uncertainty shows in the status at once; routine findings and local decisions stay inspectable without an answer. CLI status and question inspection suffice for the first version.

Configuration needs negative cases for exact overrides versus task restrictions, ceilings, unknown Packages and profiles, invalid reserves, protected inputs, conflicting instructions, oversized contexts and unknown approval Packages. Experience hints and an operational doctor are later extensions ([north star](manager-realization-north-star.md#later-operations-and-extensions)).
