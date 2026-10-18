# Configuration assignments and Manager mandates

This contract owns the execution input fields and their deterministic resolution. The [execution contract](manager-apply-execution.md) owns budget admission, run transitions and status meaning; the [journal](manager-apply-journal.md) owns their durable records.

This is a target contract. On 17 October 2026 the owner accepted project team defaults, named reviewers, conflict arbitration and complete product budget defaults ([DEC-043](../../concepts/register.md#dec-043-project-teams-review-arbitration-and-product-defaults), [source](../../concepts/sources/configuration-teams-defaults-20261017.md)). This explicitly extends the earlier single-reviewer contract and the concept freeze for this subject only. It does not claim executable support: the current v1 reference below remains a separate decoder fixture. Production schema/version migration, runtime profile composition and multi-reviewer execution are implementation work.

The observation behind this change is the Library walkthrough: repeating all finite limits and creating a profile for every model/effort combination obscured the few project choices the owner wanted to make. The selected simplification keeps team defaults and exact exceptions in the project, and moves omitted limits to captured product defaults. A distinguishing outcome is a short Library file whose fully resolved team and limits can be explained before dispatch. The extra arbitration role runs only for disputes and consumes the existing budget.

## Give each source one responsibility

| Source | Maintained content | Library example |
| --- | --- | --- |
| Canonical model | Intended behavior, required outputs, architecture and binding interfaces. | `loan-policy.spec.maxOpenLoans` is five in M1 and three in M2; existing loans remain returnable. |
| `.markitect/execution.yaml` | Manager model/effort and instructions, named reviewers and reviewArbiter, creation areas, explicit limit overrides, Package overrides, scope selection, check selection and owner approval. | Lending overrides its Manager model and reviewer settings; omitted limits use product defaults. |
| Product strategy defaults | Complete finite strategy defaults shipped with the product, with their original bytes and source fields. | Absent rejection-limit configuration takes three from the strategy document. |
| `.markitect/ignore.yaml` | Optional exact-file accounting exceptions with reasons. | One observed library editor-state file is ignored. |
| Explicit Orchestrator profile file | Literal executables, pinned image/protocol/tool identities, provider access and capabilities, resources, credential references, controlled environment and instruction inputs. Supplied by the operator; no implicit discovery. | The selected Codex build and the local test executable. Credentials are resolved locally and never copied into model or journal payloads. |
| Assignment record | The owner's goal, constraints, granted caps and recorded extensions. | Apply the current library model. |
| Registry | Versioned artifact ownership and Definition links in `.markitect/registry/`. | Lending owns a particular source file; a parent cannot acquire it through a prompt. |
| Orchestrator operation contract | Artifact-only Apply, required independent review, source and candidate bindings, guarded publication. | Configuration cannot disable review or authorize model edits. |
| Mandate section | A computed, source-bound explanation of the above for one Manager, task and role, rendered into its context package. | Lending may choose its implementation within the architecture and coordinates through its parent. |

The model stays language v1. None of these runtime records adds a canonical Assignment, Manager, Permission or Question Kind. Required behavior is never copied into execution settings; the loan limit in particular has no configuration override.

## Project execution file

One versioned YAML file outside the Definition roots describes project execution. The accepted target below needs only team choices, exact Manager exceptions and project-specific checks. Model names and instruction paths are illustrative. The `yml` target example is not a current v1 payload; its final format version is a producer deliverable, not an invented supported API version. The existing `recursive-apply-v1` strategy identity remains explicit in the run policy. The [implemented v1 reference](#implemented-v1-reference) retains the currently consumed example.

```yml
defaults:
  manager:
    model: dev-model
    effort: high
    instructions: [instructions/development.md]
  reviewers:
    correctness:
      model: review-model-a
      effort: high
      instructions: [instructions/review-correctness.md]
    maintainability:
      model: review-model-b
      effort: medium
      instructions: [instructions/review-maintainability.md]
  reviewArbiter:
    model: arbiter-model
    effort: high
    instructions: [instructions/review-resolution.md]
packages:
  lending:
    manager:
      model: strong-dev-model
      instructions: [instructions/lending-development.md]
    reviewers:
      correctness:
        effort: xhigh
        instructions: [instructions/lending-correctness.md]
      maintainability:
        effort: high
      security:
        model: security-model
        effort: high
        instructions: [instructions/lending-security.md]
    reviewArbiter:
      effort: xhigh
      instructions: [instructions/lending-resolution.md]
checks:
  - id: project-tests
    profile: library-tests
    directory: realization
    args: [test, ./...]
    requiredBy: library/callable-interface
    timeout: 2m
```

### Resolve the team

`defaults` applies to every local Package Manager, including root. `packages` keys are exact local Package paths; `.` is root. Unknown or imported Package keys fail. There is no inheritance from ancestor overrides: `lending/reminders` uses project defaults unless it has its own entry.

The accepted later [model-migration workflow](commands.md#model-restructuring-as-an-explicit-migration) explicitly migrates affected Package bindings and reviews their effective settings before activating the future Manager tree. Conflicting teams, instructions, limits or rights require resolution, not an implicit winner or added budgets. This preserves the resolution rules below; it implements no model/configuration migration API and grants no new execution authority.

| Setting | Resolution |
| --- | --- |
| Manager and reviewArbiter | Project `model` and `effort` are required. An exact Package override replaces only supplied scalar fields. |
| Named reviewers | Stable names identify entries. Missing Package entries retain project reviewers; equal names override only supplied fields. A new name needs its own model and effort. |
| Reviewer activation | `enabled` defaults to `true`. An exact `enabled: false` disables that reviewer. Every effective Manager team must retain at least one active independent reviewer. |
| Instructions | Missing lists are empty. Append project instructions for that exact role/name, then the Package additions. An explicit empty addition removes nothing. A new reviewer has no matching project list. |
| Other Manager settings | `stepTimeout`, `rejectionLimit` and `creationRoots` replace the project value within the effective ceilings, as before. |

These are explicit field rules, not generic deep merge, wildcard overrides or environment interpolation. Duplicate and unknown keys, unresolved model/provider mappings, unsupported model/effort combinations and missing instruction files fail before dispatch. Reviewer names label perspectives; they confer no extra authority and cannot restrict required obligation coverage. Model families may differ; this remains a hypothesis about useful perspectives, not a demonstrated advantage.

Instructions identify exact repository-relative regular files. Preserve order, repeats and captured original bytes; expand no directories or globs and do not normalize or deduplicate paths. Missing captured bytes require new complete acquisition and remain `NotCompleted`. Instructions contain operational guidance only. Binding project rules belong in the model ([DEC-033](../../concepts/register.md#dec-033-binding-intent-lives-only-in-the-model)). Parent messages cannot override configuration or the static operation.

| Effective role | Root and other unchanged Packages | Lending |
| --- | --- | --- |
| Manager | dev-model, high, development.md | strong-dev-model, high, development.md then lending-development.md |
| correctness | review-model-a, high, review-correctness.md | review-model-a, xhigh, review-correctness.md then lending-correctness.md |
| maintainability | review-model-b, medium, review-maintainability.md | review-model-b, high, review-maintainability.md |
| security | Absent | security-model, high, lending-security.md |
| reviewArbiter | arbiter-model, high, review-resolution.md | arbiter-model, xhigh, review-resolution.md then lending-resolution.md |

All instruction filenames in this table are below `instructions/`. The arbiter is configured but starts only for a named dispute. Managers still derive from the canonical topology, not a second authored organization. Sequential writing, guaranteed participation, independent local/integration review and the deterministic final evidence check remain mandatory.

## Strategy defaults and their sources

`limits`, `assignment` and `completionReserve` may be omitted entirely or field by field in the target format. Omitted values come from one complete, versioned product strategy document. The following owner-accepted starting values are not measured sufficient capacity and do not establish a monetary ceiling.

| Capacity | Assignment default | Hard project ceiling | Completion reserve included in the assignment |
| --- | --- | --- | --- |
| Prepared runs | 3 | 5 | None |
| Session starts | 80 | 160 | 24 |
| Agent steps | 160 | 320 | 40 |
| Active execution time | 8h | 16h | 2h |

| Effective setting | Product default |
| --- | --- |
| `defaults.stepTimeout` / `limits.maxStepTimeout` | `10m` / `20m` |
| `limits.hostOperationTimeout` | `2m` per Orchestrator control operation |
| `defaults.rejectionLimit` / `limits.maxRejectionLimit` | `3` / `3` |
| `limits.maxContextBytes` | `400000` |
| `limits.maxSessionSteps` | `40` |
| `limits.maxTransientRetries` | `2`, only after proven non-dispatch |
| `defaults.creationRoots` / `limits.creationRoots` | `[realization]` / `[realization]` |
| Instructions for each role/name | `[]` |
| `defaults.batchNoChangeReviews` | `false` |
| `acceptance.ownerApproval` | `[]` |

No universal concrete model, provider credential, team perspective or project check command is invented by the product. Models/effort and the review team are explicit project choices; operator access and capabilities remain explicit local inputs. An authorized project override can change a ceiling; it is not extra capacity automatically granted to an existing assignment.

Retain absence in authored input. Resolve product defaults, explicit project values and exact Manager exceptions; a concrete assignment may explicitly select caps within the effective project ceilings. Capture original product bytes and digest, using `product:recursive-apply-v1/defaults.yaml` with the actual field pointer as provenance. Missing required defaults leave preparation `NotCompleted`; explicit project values retain their actual source. There is no silent clamping or fabricated execution-field source. The run binds the strategy version, defaults bytes, every effective value, source and role/profile identity. Product updates cannot change an existing run; preparation for a new run exposes changed effective defaults and their source.

The reserve is part of the caps, not extra allowance. Validate every effective value, ceiling, creation area, reserve and the complete configured review census before preparing a run. A smaller explicit cap conflicting with a product-default reserve fails with both sources identified. For example `maxRejectionLimit: 2` with an omitted rejection limit fails because its effective default is three. Budget extensions are explicit recorded grants, never an automatic increase to the ceiling. The [execution contract](manager-apply-execution.md) owns accounting and finite exhaustion, including arbiter work.

## Resolve settings deterministically

The [team field rules](#resolve-the-team) and [product-default resolution](#strategy-defaults-and-their-sources) are the only configuration precedence rules. Root overrides configure root only. Before dispatch, expose each Manager's effective model/effort, named active reviewers, arbiter, ordered instructions and limits with file/field/digest provenance. A rendered view explains the bound settings; it grants no authority and contains no credential values.

`scope` still optionally selects Kinds/Properties whose local Package paths determine scope; reject unknown paths. Configuration does not enumerate individual Definitions. A project check names its model obligation through `requiredBy`, which must resolve. Literal command repetition is the accepted exception to single intent ownership; review keeps it aligned with the model. Checks still use confined candidate directories and literal arguments, never shell strings.

Creation areas grant neither other Managers' files nor protected inputs. The Library default `realization/` can be explicitly changed in a project's ceilings and selections; ownership, reservation and input guards still apply. No config field can disable required participation, review, final evidence, coverage or guarded publication.

## Implementation boundary

This documentation delivery changes the accepted target, not Go decoders, executable defaults, runtime profiles or fixture inputs. Follow-up implementation must version the execution/defaults formats and migrate preparation, role resolution, rendering, protected instruction acquisition and run bindings together. Use the existing profile composition to turn direct role model/effort into adapter-specific admitted profiles; reject ambiguous or unsupported mappings. Do not add a second profile catalog, implicit provider discovery, user override layer or model-language Kind.

The [core](manager-apply-core.md#manager-tasks) and [execution](manager-apply-execution.md#review-conflicts-and-arbitration) own multi-reviewer acceptance and arbitration; [ports](manager-apply-port-contracts.md#configured-review-team-bindings) and [journal](manager-apply-journal.md#review-team-and-arbitration-records) own the necessary bindings. Producer APIs and adapter/CLI/bootstrap changes remain coordinated implementation deliveries. These contracts do not authorize Docker, providers, credentials, Apply, comparisons or studies.

## Implemented v1 reference

The following older single-reviewer example remains the current decoder/test reference, with its original values. In particular its `maxStepTimeout: 10m` is not the newly accepted target ceiling of `20m`. Current production defaults supply only a subset of the complete target table. Do not interpret passing this fixture as support for named reviewers, arbitration or omitted budget blocks. Retire or migrate the reference together with its actual consumers during implementation.

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
  maxRejectionLimit: 3
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
  rejectionLimit: 3
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

<a id="host-profiles-and-operational-state"></a>

## Orchestrator profiles and operational state

The operator supplies an explicit profile file and an absolute state directory outside the governed repository; acquisition records both. The profile file defines `agentProfiles` and `checkProfiles` with typed, versioned, adapter-specific sections. Current v1 refers to them by name. The accepted target composes project role model/effort and captured instructions with explicitly selected, compatible Orchestrator runtime bindings. `run` receives prepared profile identities, not provider wire objects. Exact format/API migration is implementation work; no provider is selected from ambient configuration.

Orchestrator instruction inputs outside the repository remain explicit locations in adapter-owned configuration. Orchestrator profiles keep their inline typed, versioned, adapter-specific sections. The concrete driver fields, versions, schema and readiness checks belong to the actual consumer and adapter deliverable.

| Profile | Required preparation |
| --- | --- |
| Agent | Adapter kind (Codex App Server or Claude Code), runtime image ID, the CLI version and protocol-schema identity the image records, effective project-selected model and effort, credential reference, container and transport limits, effective provider settings and tool schemas, explicit environment and instruction policy. |
| Check | Runtime image ID, executable inside the image, fixed environment policy, argument prefix, container, output and termination limits. Project arguments are appended literally. |
| State location | Repository and store identity, a confined private directory, the SQLite journal configuration, and the container engine executable with the runtime's volume prefix. Its identity is validated before an assignment resumes. |

The selected bytes and identities enter the run's input binding and are rechecked at effect boundaries. Secrets are never persisted; capture safe configuration and instruction bytes, secret references and identity bindings, and report the observation boundary. A credential reference names an environment variable or a private file on the Orchestrator; its value goes only to the provider process inside the container ([rebuild plan](manager-apply-rebuild-plan.md#container-runtime)). The [rebuild plan](manager-apply-rebuild-plan.md#bind-the-runtime-image-before-real-work) also owns the image and handshake gate.

## Assignment run task and attempt

| Identity | Meaning and lifetime |
| --- | --- |
| Assignment | One owner request with its caps. It links its runs, questions, consumption and extensions and owns no scheduler. |
| Apply run | One immutable model, configuration and source binding and its reconciliation. A changed basis or execution policy needs a continuation run after this one finished. |
| Task | One Manager responsibility within a run. Tasks in a continuation get new IDs with predecessor links where work carries forward. |
| Attempt | One submission within a task against the run's fixed basis. It resets no budget or rejection counter. |
| Execution session, conversation and step | A bound execution environment, separately identified provider dialogue and one dispatch. In the first Apply each step has its own execution session ([S1](manager-apply-core.md#steps)), and a conversation spans many of them ([S3](manager-apply-core.md#steps)); each step binds its current workspace and authority. None supplies authority by its identity alone. |

The assignment stores the owner input and goal, constraints, operation target, project identity, caps and authority source. Omitted caps use the `assignment` defaults. Renewing an exhausted cap needs a recorded extension within the project ceilings; raising a ceiling needs an authorized configuration edit. Starting another run grants nothing. The first product has one local operator as decision owner and no identity service based on Agent-reported names. An accepted decision can authorize a continuation run within the assignment; it never authorizes merge, release, deployment or production-data edits.

## The mandate section

`organization` renders the mandate as a section of the context package, not as a separate record; `run` passes task and assignment facts as plain inputs, so organization does not import run. The section carries its source references and is bound to the task record with the rest of the package. The Manager, each named reviewer and any admitted reviewArbiter receive the applicable mandate; the reviewer uses it as evidence under its own read-only grant, never as write authority.

For lending it states:

- Responsibility: reconcile all local lending Definitions and coordinate reminders through its child.
- Binding basis: the current policy, member eligibility, copy availability, returns and required outputs, with the original passages available.
- Freedom: the internal implementation within the model's architecture and interfaces; no additional business restrictions.
- Artifact authority: its owned files plus new paths claimed through the Orchestrator; changes to other owners' files are requested through the common parent.
- Reserved choices: changed business rules, authority or protected inputs need a decision outside artifact Apply.
- Limits: the remaining assignment capacity, local step and rejection limits and the mandatory gates.

The section may display the bound loan limit but cites `loan-policy` as its authority, so M2 changes one canonical value and no mandate text. The reviewer checks material decisions against their source-bound authority as well as their behavior. The Orchestrator validates the typed boundaries; judging semantic authority remains reviewable Agent work.

## Questions and owner attention

A Manager first retrieves existing knowledge, then decides within its freedom or coordinates through its common parent. It escalates only a necessary decision beyond its mandate. The root has a broader coordination scope, not more authority over business meaning.

The [journal contract](manager-apply-journal.md#questions-answers-and-model-changes) owns the content of questions, how owner answers are bound and what they trigger. A blocked assignment or a serious execution uncertainty shows in the status at once; routine findings and local decisions stay inspectable without an answer. CLI status and question inspection suffice for the first version.

Configuration needs negative cases for exact overrides versus attempted parent-message restrictions, ceilings, unknown Packages and profiles, invalid reserves, protected inputs, conflicting instructions, oversized contexts and unknown approval Packages. Experience hints and an operational doctor are later extensions ([north star](manager-realization-north-star.md#later-operations-and-extensions)).
