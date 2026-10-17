# Manager organization and reliable Apply

This is the product concept for realizing a canonical model through Managers. It owns organization, context, delegation, deliveries, review purposes, artifact ownership, the Apply lifecycle and reporting. The [design map](README.md#apply-design) lists the technical contracts that refine it and states the status of the whole set. The [north star](manager-realization-north-star.md) keeps the ideas and variants that this concept did not select.

The concept uses the [completed language v1](minimal-language/language-contract.md) without new Core Kinds or semantics. The owner decisions include [DEC-025 to DEC-036](../../concepts/register.md#dec-025-package-managers-organize-realization) and the superseding [DEC-039 process clarification](../../concepts/register.md#dec-039-deterministic-manager-process).

The product is an organization that realizes an explicitly described project world. The canonical model supplies intended meaning and constraints. A deterministic Orchestrator organizes work, supplies context, validates deliveries and records the process. AI Managers make bounded realization judgments and integrate results. The Orchestrator propagates the operation and enforces the process; an Agent does not choose who participates or rewrite another Manager's operation. The first operation is Apply: reconcile the existing artifacts with the current model, including when the model has not changed.

The owner accepted [named review teams, conflict arbitration and product defaults](../../concepts/register.md#dec-043-project-teams-review-arbitration-and-product-defaults) on 17 October 2026. This bounded extension replaces the single-reviewer-only target; source implementation remains separately versioned. The [configuration owner](manager-apply-configuration.md) holds the team syntax/defaults; the [execution owner](manager-apply-execution.md#review-conflicts-and-arbitration) holds dispute and rejection semantics.

## Selected choices

| Question | Choice | Reason |
| --- | --- | --- |
| Organization | One ordinary root Manager and one Manager per local Package. | Responsibility follows the structure the author already maintains. |
| Code, tests and documentation | One hierarchy responsible for all artifact forms. | A feature's implementation, tests and documentation stay coordinated. |
| Default team | The Manager does its local work; the configured named independent reviewers assess each delivery and integration. A fresh reviewArbiter settles explicit disputes. | Project defaults and exact Manager exceptions preserve independent review without another Manager hierarchy. |
| Final check | The Orchestrator's deterministic evidence check after the root's integration review; no separate verifier Agent. | The root's integration review already judges the whole; the Orchestrator makes sure all evidence is current. |
| Parent knowledge | Summaries first; targeted, recorded source inspection when needed. | Local responsibility without depending entirely on child claims. |
| Context | Own Definitions and contracts, explicit neighbors, Definitions whose modeled scope covers the Package, and task facts. | Simple, inspectable compilation without semantic retrieval. |
| Order | The Orchestrator propagates the fixed operation down every branch, then runs ready work, dependency responses and upward integration under a fixed protocol. | No Agent can omit a responsibility or bypass its review. |
| Acceptance | Apply starts from the committed model and writes its verified result into the working tree; the owner's commit accepts it. | Acceptance stays with the owner, as for the model. |
| First reconciliation | Every local Manager receives the operation and assesses its area; no-change results receive review too. | Participation never depends on root relevance judgment or incremental invalidation. |
| Execution | One step at a time ([S5](manager-apply-core.md#steps)), in owned candidate workspaces. | Reliable integration before concurrency. |
| File ownership | One accountable Manager per artifact; many-to-many links between artifacts and Definitions. | Write responsibility is separate from what an output realizes. |
| Prompts and history | Record the actual inputs and observed events; derive readable reports from them. | Work is inspectable without sending every transcript to every Agent. |
| Other commands | Later operations over the same organization. | The first delivery concentrates on Apply. |

## Four kinds of input and record

| Concern | Owner and meaning |
| --- | --- |
| Canonical model | Authored Definitions, contracts, bindings and relationships: what should hold. |
| Organization configuration | Project-owned execution choices: teams, profiles, limits, instructions and context presentation. It cannot redefine canonical meaning ([configuration](manager-apply-configuration.md)). |
| Realization registry | Tool-maintained, versioned metadata on artifact ownership and justified links to Definitions. It describes the actual realization, not a second intended world. |
| Execution history | Immutable records of inputs, communication, candidates, reviews and outcomes ([journal](manager-apply-journal.md)). |

Binding intent lives only in the model ([DEC-033](../../concepts/register.md#dec-033-binding-intent-lives-only-in-the-model)). The test is simple: if violating something makes an artifact wrong, it is model content. That includes realization constraints such as a public interface, the programming language or required build and test commands; the library's [callable interface](library-example/m1/model/library/callable-interface.yaml) is an example. Configuration and instructions hold only execution choices and operational guidance, such as profiles, limits, reporting style or tool usage. A reviewer who finds a project rule in configuration or custom instructions records a finding, and the rule moves into the model.

Configuration and registry stay outside the canonical Definition roots; v1 would otherwise require a Definition envelope. The compiler produces a bound structural model and diagnostics and does not interpret prose as rules. The Orchestrator applies deterministic workflow rules to explicit snapshots. Agents interpret meaning and produce reviewable work. No prompt or report can change the operation target or authorize a model edit during Apply.

## Authoring and inspectable preparation

A small optional software vocabulary and the complete [library example](library-example/README.md) are ordinary, versioned models. Derive the vocabulary from the example's needs instead of building a universal catalogue. It has no privileged compiler behavior; authors can use their own terms, replace it or extend it. Templates produce visible, editable Definitions in the uniform YAML form, not a second authoring language.

Before Agents start, the author can preview what each Manager will receive: which statements, contracts, responsibilities and explicit connections, and why each is included. Original model content, deterministic explanations and later Agent judgments stay distinguishable. The preview exposes mistaken context and missing connections; it cannot establish semantic completeness. Preparation is part of every Apply; there is no requirement to precompile contexts for every possible task.

Add structure only when it exposes a concrete failure possibility or saves recurring human work. Prefer clearer instructions and views over new roles or parallel records.

## Organization and configuration

The root Manager follows the same protocol as every other Manager. It has its own runtime identity for the whole selected local model; v1 forbids a root Package, so none is invented. Top-level Package Managers are its children, and parents follow folder ancestry. Every Manager receives the operation, including an empty leaf or an intermediate Package with one child; there is no pass-through startup exception in this strategy. An empty local Definition set is an explicit assessment result, not permission to skip that Manager or its existing ownership and child responsibilities. Insufficient capacity leaves the run incomplete rather than silently omitting a Manager. The run binds the derived topology as execution metadata; empty folders stay out of the canonical model digest as v1 specifies.

Imported models supply read-only vocabulary and context and create no Managers. Editing a library is a task in that library's own project. One local model is the first supported organization.

Configuration has project defaults plus exact Manager overrides and separate hard ceilings. An override for lending does not configure lending/reminders. The Orchestrator enforces model obligations, resolved configuration and fixed process constraints; parent hints cannot add task restrictions or narrow a child's responsibility. The [configuration contract](manager-apply-configuration.md#resolve-settings-deterministically) owns resolution. Configuration cannot disable required delivery or integration review, the final evidence check, coverage or source-drift guards.

| Configurable part | First design |
| --- | --- |
| Team | Manager plus at least one named independent reviewer, and a configured dispute-only reviewArbiter. The earlier single-reviewer source is not multi-reviewer support. Implementer roles, reviewer per Definition and separate artifact hierarchies remain later strategies. |
| Execution profiles | Project role model/effort and instructions with exact Manager overrides; explicit compatible Orchestrator inputs supply provider/runtime/resources and credential references. |
| Scope of shared rules | The model states a rule's scope through a Property whose values are local Package paths, for example `appliesTo: [lending]`, or `[.]` for the whole model. Configuration only names which Kind and Property carry scope; it never lists individual Definitions. |
| Owner approval | Optional list of Packages whose changes wait for the owner's approval before publication. |
| Review batching | Optional: one reviewer session may assess several no-change deliveries, each with its own bound verdict. |
| Artifact creation | Permitted creation areas and exclusive reservation of new paths. Artifact paths need not mirror Package paths. |
| Limits | One shared assignment ledger and a finite rejection allowance, owned by the [execution contract](manager-apply-execution.md). |
| Concurrency | One step at a time ([S5](manager-apply-core.md#steps)). Later parallelism must use the same reservation and integration rules. |

### Project-level artifacts and the skeleton

Ownership follows creation: the Manager that reserves a new path owns it. For a project-level artifact, the requiring Definition decides who creates it. In the library, `library/callable-interface` requires the Go module, the entry point and the build and test commands, so the `library` Manager creates and owns them. When Definitions in several Packages require the same artifact, their nearest common parent decides the owner and records the decision; that can be the root. A file that no Definition justifies is never owned: it is a Markitect input or explicitly ignored, so CI configuration, for example, needs a Definition before Apply manages it.

When a required skeleton is missing, its owner can supply a limited normal delivery while dependent work waits: module, entry point and the conventions the model requires. Its exact usable scope and outstanding work remain explicit; Orchestrator acceptance of this prerequisite does not complete that Manager or subtree. The consumer waits until the delivery is reviewed and accepted; its next step then sees it in the aggregate ([D3](manager-apply-core.md#deliveries-and-composition)). The same owner later integrates the completed operations into its entry point. These use ordinary owned work and review; the root does not invent a global delivery plan, and the Orchestrator does not infer an execution order from arbitrary references.

## Tasks and delegation

Apply is one static, versioned operation instruction: reconcile the artifacts of your responsibility with the bound model, assess existing realization, correct deviations and explain the result. It is combined with each Manager's separately compiled context, resolved configuration, role template and current Orchestrator facts. The instruction enters through the root and reaches every local Manager unchanged. Questions, findings and delivery messages carry current work facts; they cannot replace, narrow or rewrite that instruction. The model remains read-only.

The Orchestrator records every parent-child handoff and guarantees recursive propagation to all direct children. It does not depend on an Agent calling a delegation tool, returning a participant list or remembering a child. Logical receipt and physical session dispatch stay distinct: a stored prompt is not proof the Agent received it. Each Manager acts only in steps the Orchestrator admits ([T4](manager-apply-core.md#manager-tasks)), and none of its containers runs between steps ([S1](manager-apply-core.md#steps)). Admission order cannot skip a Manager.

The organization metaphor has a precise boundary: Managers own local realization and the integration of their direct children's results; the Orchestrator owns mandatory participation, permitted routes, required reviews and completion. The root is an ordinary integration Manager, not an AI workflow planner. A parent retains its own artifacts and can resolve local prerequisites before its children finish.

| Fixed process responsibility | Agent judgment |
| --- | --- |
| Compiler pipeline derives hierarchy, explicit context and inclusion reasons. | Interpret the supplied obligations in the local realization. |
| Orchestrator propagates the same operation to every local Manager. | Return a justified change or no-change result. |
| Orchestrator admits ready work one step at a time and enforces finite budgets. | Report concrete prerequisites, questions and corrective work. |
| Orchestrator routes every message along tree edges and records each hop. | Explain a question or proposed agreement with its model basis. |
| Orchestrator requires local and integration review and current evidence. | Independent reviewer judges whether realization and evidence suffice. |

### Fixed Manager cycle

1. **Receive and propagate.** The Orchestrator binds the operation and per-Manager context, records receipt and propagates the unchanged instruction to every direct child. Each Manager remains in the completion census. A failed or uncertain delivery cannot count as completed participation.
2. **Explore before writing.** Every Manager examines its own obligations and current artifacts without changing them, and reports proposed changes, prerequisites, uncertainty and open questions. Explore runs children before parents, so each parent combines its direct children's reports into a broader view ([T1](manager-apply-core.md#manager-tasks)). A Manager that finds nothing to change submits its no-change result directly ([T3](manager-apply-core.md#manager-tasks)). No artifact writing starts until every task is explored ([T2](manager-apply-core.md#manager-tasks)). Missing or failed participation blocks that barrier; an assessment may report unresolved work. Explore reduces late surprises but does not prove that all dependencies are known.
3. **Plan locally, work and review.** Each Manager proposes its own realization work within the unchanged mandate. There is no global Plan barrier or root-authored delivery plan. After Explore, work becomes ready when its local plan and concrete prerequisites permit it; unresolved dependencies hold only the work that needs them. Each work step ends with `submit` or `wait` ([T5](manager-apply-core.md#manager-tasks)). The active owner writes only its allowed artifacts. The Orchestrator observes and seals the candidate. Independent local review judges changes or a no-change proposal with the required checks. A normal delivery may have a limited usable scope while its Manager remains open: independent review and Orchestrator acceptance must precede dependent implementation, whose next step then sees the delivery in the aggregate ([D3](manager-apply-core.md#deliveries-and-composition)). Review is required during handoffs, not only at the end.
4. **Integrate upward.** Once every direct child is done and local work is accepted, the parent's integration step receives the composed aggregate of its subtree with summaries and evidence and judges their interaction ([T9](manager-apply-core.md#manager-tasks)). Parent-owned integration changes receive ordinary review. A fresh integration review judges the aggregate; one-child parents still participate.
5. **Correct through the tree.** A finding is an issue addressed to its responsible owner ([I1](manager-apply-core.md#issues), [D5](manager-apply-core.md#deliveries-and-composition)). Dependent work waits for the required reviewed result. Changed content or assumptions require renewed affected assessment; old acceptance does not carry to a different subject. Unknown affected scope broadens inquiry and review, never narrows participation.
6. **Complete and publish.** A Manager's 'done' is a proposal. The Orchestrator accepts a subtree only after its local and descendant obligations, current required reviews/checks and blocking questions are complete. At the root, final evidence, source preflight, any required owner approval, guarded publication and read-back remain mandatory. All Agents agreeing does not establish success by itself.

These are steps within the existing run lifecycle, tracked by the [task machine](manager-apply-core.md#manager-tasks), not another run state machine. In the first Apply, steps run one at a time ([S5](manager-apply-core.md#steps)). Later discoveries reopen affected planning or assessment without restarting the initial global Explore barrier or stopping independent branches.

### Ready work and parent guidance

The Orchestrator records when work becomes ready and admits it in the order [T4](manager-apply-core.md#manager-tasks) fixes. Eligibility includes current inputs, resolved prerequisites and budget. A blocked task does not prevent other ready tasks from running, and becoming ready again records a new queue entry without erasing its earlier wait. This is deterministic for the same recorded event sequence; different provider response times may produce different sequences.

A dependency wait is a `wait` outcome that names the missing accepted result or decision ([T5](manager-apply-core.md#manager-tasks)). Because no container runs between steps, a waiting task holds no write access; artifact ownership remains unchanged, and uncertain writer effects are reconciled before the workspace admits another step ([S2](manager-apply-core.md#steps)). Mutual waits go to the common parent for bounded clarification ([T12](manager-apply-core.md#manager-tasks)), not repeated polling or guessed ordering. A necessary missing model relationship still requires a separate model correction.

Parent hints are separate, attributed messages alongside the unchanged operation, compiled context and current Orchestrator facts. They may point out an integration concern or suggest an implementation approach within the model's freedom. They cannot create binding project rules, narrow another Manager's responsibility or waive a required check. Parents work from summaries of results, interfaces, prerequisites, findings, uncertainty and evidence; they request targeted detail when needed rather than reading every descendant transcript.

The owner confirmed a bounded, versioned direct-child report ([confirmation](../../concepts/sources/manager-process-clarification-20261017.md#bounded-coordination-and-idle-execution)). Its [message contract](manager-apply-port-contracts.md#bounded-child-reports) distinguishes initial assessment, limited delivery and completed responsibility, binds the exact step or delivery, and preserves prerequisites, findings, uncertainty and evidence references. A per-child limit alone is insufficient: the assembled parent input also stays within its context limit. Visible partitioning and targeted retrieval retain every required child's status and outstanding obligations; omitted detail is never agreement or completion. Summaries supply Agent judgments, not Orchestrator acceptance.

### Communication and dependencies

Messages travel only along tree edges, and the Orchestrator records every hop ([M1](manager-apply-core.md#messages)). A question to a known Manager travels up to the common parent and down the appropriate branch; the Orchestrator forwards it without dispatching the Managers in between ([M2](manager-apply-core.md#messages)). A model link can add compiled context but never grants a communication shortcut or write ownership.

Name the addressee from explicit responsibility and model facts when possible. When the target is unknown, ask the parent, which replies or names a better addressee in its subtree ([M3](manager-apply-core.md#messages)). The Orchestrator records the question, its hops and outstanding replies, so silence is not agreement. Inquiry seeks responsibility or an implementation fact, not an inferred missing model relationship. Apply cannot add a missing link or compensate by inventing context; a necessary missing relationship requires a separate model correction and blocks the affected assessment.

A reference is not automatically an execution dependency. A concrete dependency states the needed reviewed result and its basis. Its owner may work while dependents wait, including a parent supplying a skeleton. Managers retain local API ownership; consumers request changes through the tree and never edit producer files. Unresolvable ownership, circular waits or conflicting requirements remain visible decisions or incomplete work, not an endless exchange. Inquiry and correction consume the existing finite budget and cannot reset review counters.

### Remaining process detail and technical boundary

The owner confirmed Explore, local planning, recorded readiness, separate parent hints and limited normal deliveries on 17 October 2026 ([source](../../concepts/sources/manager-process-clarification-20261017.md#explore-planning-and-limited-deliveries)). These choices address late discovery of shared prerequisites without returning participant selection or workflow design to AI. The library trace must show every initial assessment before the first write, an accepted skeleton unblocking a consumer while its producer stays open, and independent work progressing around waits.

Review reuse requires an unchanged complete bound subject, including its relevant inputs and assumptions. An Agent's claim that a changed file is unrelated is insufficient. Changed bindings require renewed affected assessment; unknown effects widen it. Whether a local review survives composition onto a newer aggregate is decided by [D4](manager-apply-core.md#deliveries-and-composition). An altered aggregate needs current integration evidence even if a local subject is unchanged. Fine-grained semantic reuse is not required for the first Apply.

The owner also selected a review rejection limit: successful limited deliveries consume shared execution capacity but neither use nor reset rejection allowance. Rejections accumulate per run, Manager and review purpose; after two, the parent assesses the repair plan, and the default third rejection leaves the task incomplete, which finishes the run Incomplete ([T14](manager-apply-core.md#manager-tasks)). This replaces the earlier submission-count design, which could exhaust its limit on successful handoffs alone. The [execution contract](manager-apply-execution.md#admission-amendments-and-finite-correction) owns exact counting and exhaustion. Before recursive implementation, define and validate the technical message, readiness, delivery and counter bindings against this process. No second planner, reviewer hierarchy, semantic index or parallel workflow ledger is needed.

The compiler pipeline here means deterministic preparation over compiled model facts. The existing `model`/`organization` separation can provide it; no new Core Kind or model-language change is implied. Managers and reviewers still interpret prose. The compiler can reject unresolved references but cannot prove that every necessary semantic relationship was authored.

### Working bases and conversation continuity

The owner selected continuity across limited deliveries rather than a fresh conversation for each changed working basis ([confirmation](../../concepts/sources/manager-process-clarification-20261017.md#working-bases-and-session-continuity)). The earlier contract fixed a candidate to the whole execution session; it would unnecessarily discard context when a Manager needs a new accepted prerequisite. The first Apply realizes this choice in a step-bound form: each step runs in its own container ([S1](manager-apply-core.md#steps)) and continues the Manager's conversation or receives an Orchestrator-built handoff ([S3](manager-apply-core.md#steps)), and nothing changes a workspace inside a running container ([S6](manager-apply-core.md#steps)). The target separates four lifetimes:

| Subject | Lifetime and authority |
| --- | --- |
| Run inputs | Immutable captured model, configuration and original repository inputs. A model or configuration change still requires the existing stop/continuation process. |
| Manager working basis | The accepted aggregate current at the step's admission ([D3](manager-apply-core.md#deliveries-and-composition)). Every later step sees each newly accepted delivery. |
| Own working changes | The Manager's unaccepted changes, recorded after each writing step ([S4](manager-apply-core.md#steps)) and replayed onto the current aggregate by the D2 rule. They are neither an accepted dependency nor an approved result. |
| Review candidate | An immutable snapshot and its exact review purpose, inputs and evidence. Later working changes cannot alter that subject or inherit its approval. |

A stable Manager identity and conversation span many steps. Each step records the aggregate version it saw, and its file edits are observed outputs. Changed assumptions reopen affected assessment; unknown influence broadens it.

The next message identifies the current aggregate version, accepted input changes, preserved own changes, open findings and relevant reassessment. The Manager checks changed interfaces and assumptions from current sources. Old conversational claims are context, not current evidence. New dialogue, a container replacement or a context handoff cannot reset responsibility, findings, budgets or rejection counters.

Continuation stays within the existing session-step and context limits. Use bounded summaries and retrievable evidence rather than copying every transcript into each step; never silently omit mandatory current obligations or findings to fit. An oversized required context remains a visible refusal. Provider-internal compaction or unavailable history is an observation limit, not a claim that all earlier context was preserved.

Demonstrate the library case: lending keeps its own unfinished work when a catalog delivery is accepted, sees that delivery in its next step's aggregate and reassesses the affected interface; a crash during materialization cannot cause an unrecorded continuation or a repeated effect ([S2](manager-apply-core.md#steps)).

The owner also selected bounded idle execution, not one permanently running container per Manager. A logical Manager remains in the census while its conversation and task evidence are retained without a running environment. In the first Apply the warm limit is zero, so every step is a charged session start ([S6](manager-apply-core.md#steps)); the [idle policy](manager-apply-execution.md#idle-execution-and-resumption) keeps its accounting.

These refinements address the owner's scaling concern without replacing the first Apply's complete participation. An observable library case delivers a limited catalog report without declaring catalog complete, exposes all pending lending prerequisites even when reports are partitioned, and resumes lending in a new container with the same responsibility and counters. Reuse of earlier Manager assessments and coordination-based modeling suggestions remain [later extensions](manager-realization-north-star.md#later-operations-and-extensions), not exceptions to initial Explore or required evidence.

### Fixed-source comparison and improvement criterion

Comparison basis: Main [`0392cd2f`](https://github.com/Glacius-Labs/Markitect/tree/0392cd2fd724d98b3fc715cb5fd3db22e2d5d224), 17 October 2026. This is an assessment snapshot, not a live backlog. The owner observed that AI-created child prompts and root planning reintroduced discretion over already explicit structure ([source](../../concepts/sources/manager-process-clarification-20261017.md)).

| Surface at that source | Assessment and required follow-through |
| --- | --- |
| `internal/organization/select_context.go`, `topology.go` | Explicit selection and direct parent/child inventory fit. The `PassThrough` startup assumption and empty-leaf omission must not become scheduling policy. |
| `internal/organization/render_templates.go`, `render_context.go` | Bounded deterministic rendering fits; prompts to 'delegate outcomes' and arbitrary supplied task outcomes must become the static operation plus separately identified current messages. |
| `internal/run/binding.go` review-floor calculation | It excludes empty local work and pass-through integration. Reassess the reserve against mandatory participation and the corrected required reviews. |
| `internal/run/session*.go`, `state.go`, `restore*.go` | Durable sessions, sends, budgets and uncertainty are useful foundations. They do not yet implement the Manager message protocol or recursive completion. Preserve their effect observations. |
| PR329, despite its branch name `dev/run-local-review-ledger` | Its integrated change records complete versioned basis inventories. It is not a local-review or delegation engine. |
| `internal/apply/control*.go`, snapshot/realization and container/agentd foundations | Control, exact candidates, ownership and physical isolation remain applicable. This decision requests no runtime rewrite. |
| Earlier design `plan` / `delegate` tools and parent-created prompts | Superseded by this process. The [core protocol](manager-apply-core.md#host-tools) defines the Orchestrator tools; old illustrative APIs are not implementation authority. |

Observable improvement: a controlled trace delivers the identical operation instruction to every Manager with separately compiled context even when the root omits a child in its output; forwards cross-area questions only along tree edges, recording every hop; records reviewed no-change results; services a parent prerequisite without deadlock; and refuses subtree success while a child, review, dependency response or changed-subject assessment remains open. Real provider execution and the two real Applies remain separate evidence.

## Context packages

Each context package is an immutable rendered input plus source references. Its bytes and the actual outgoing invocation are recorded. Context compilation is deterministic for its explicit inputs; later semantic conclusions are attributed Agent judgments. No Agent selects initial relevance or repairs missing model relationships by adding inferred context.

The initial Manager package contains:

1. Its task and role, the mandate (responsibility, binding basis, freedom, artifact authority, reserved choices and limits, as the [configuration](manager-apply-configuration.md#the-mandate-section) renders it), effective configuration, operation target and review requirements.
2. Every local Definition in its Package, in stable path order: the filename as heading, then identity, Kind, complete body and structured `spec`, plus the bound Kind and Property contracts.
3. A See also section: direct outgoing references and body links, and direct incoming references to the Package, each with reason and direction. It includes the directly connected Definitions and their type contracts; further references stay available to inspect.
4. Every Definition whose modeled scope covers this Package, because its scope lists the Package, one of its ancestors or `.`, as a read-only view of the bound Definition with its reason for inclusion. It does not extend import resolution.
5. A compact local Package directory for routing questions, and a child index with identities, purposes, task states and latest validated reports. It does not load foreign Definitions, artifacts or transcripts.
6. The Manager's artifact inventory and relevant candidate files for local work. A Manager that integrates children first receives their delivery summaries, agreements and evidence references.

A rendered package larger than the configured limit stops preparation with a finding to split the Package; a package is never truncated silently. The role prompts for root, Manager and reviewer are product-owned, versioned templates. Their identity is recorded with every invocation, changes to them go through review, and custom instructions are only appended.

An implementer receives the local task and candidate inputs. A reviewer works in fresh context without the implementer's conversation: its first step receives the task and the relevant model contracts only and records the expected behavior and discriminating cases; the second step supplies the actual candidate and evidence. The Orchestrator checks that a reviewer's rendered package contains what its role requires; the reviewer judges whether that context suffices.

Managers can request further Definitions, reports, transcripts or artifact files by identity; the request and the returned content are recorded. A coordinating parent is not loaded with descendant source files initially but can inspect them to resolve a conflict or judge integration. Need to know keeps Agents focused; it is not an operating-system isolation boundary.

Normal context discovery follows explicit references. A body link marks a connection, not proof that every obligation at its target applies. A missing necessary connection is a modeling defect to surface, not permission to ignore a known obligation. Apply never invents canonical links. A necessary missing link blocks the affected assessment rather than being compensated with inferred Apply context. The optional Optimizer belongs to a separate model operation ([north star](manager-realization-north-star.md#missing-links-and-the-optimizer)).

## Checking effects across Package boundaries

During local assessment, the Manager and its reviewer both ask: which statements in this area require behavior from another area, and is that connection visible in the model? For example, members states that suspended members cannot borrow, while borrowing belongs to lending. Reading the local statement is enough to raise the question; no semantic search is required.

Raise an ordinary open question naming the source Definition and passage, the possibly affected Package and the suspected effect, and address it to that Package's Manager; the Orchestrator forwards it through the nearest common parent ([M2](manager-apply-core.md#messages)). If no target is known, address it to the parent without inventing one ([M3](manager-apply-core.md#messages)). Apply can report the gap but cannot add the link; a model correction is a separate model operation followed by a new Apply. An unresolved question that prevents a sound assessment blocks completion of the affected work and therefore of the aggregate.

This is a review duty over the statements actually assessed, not a guarantee that every missing connection is found.

## Technical risk derivation

Managers and independent reviewers must derive relevant technical failure modes from the applicable model obligations, available operating conditions and actual realization. The owner need not enumerate every implementation risk or author a separate Definition for every technical consequence of an existing obligation. Managers perform this investigation during Explore and continue it during implementation, rework and integration; each reviewer independently examines the local or composed subject. A no-change result has the same duty.

Ask how the realization could violate an obligation under the conditions in which it is intended to operate. Relevant concerns can include concurrent updates, the interval between checking a precondition and committing, partial failure, retries, lifecycle transitions and external effects. These are examples to select against the actual responsibility, not an exhaustive checklist or an instruction to invent operating conditions. For example, when concurrent use is in scope, an obligation that publication freezes content requires examining a write that read editable state before another operation published it. A green sequential test alone does not settle that concern.

Deriving a necessary technical consequence of existing intent is realization judgment within the mandate. Managers investigate, choose a conforming implementation and repair deviations; reviewers judge the reasoning and evidence. A different business policy, an exceptional tradeoff outside the mandate or a necessary missing model relationship still follows the existing question and separate model-change route. Technical inference does not authorize invented canonical links, new business restrictions, unsupported runtime assumptions or additional execution rights.

The two review stages remain unchanged. Before candidate access, reviewers derive expectations and distinguishing cases from the task and model. After candidate access, they must inspect the actual implementation and integration for additional risks and reassess whether the original cases suffice. The first-stage expectations remain recorded; additional concerns and needed cases use the existing findings and evidence path in the [execution contract](manager-apply-execution.md#evidence-need-survives-budget-pressure). Managers retain the grounds, investigation and unresolved concerns in their existing reports and Definition assessments. No separate risk register, role, workflow stage or schema is introduced.

The [owner clarification of 17 October 2026](../../concepts/register.md#dec-045-managers-and-reviewers-derive-technical-risks) addresses the gap between handling a recognized concern and actively discovering it. The observable improvement is a Manager or reviewer deriving a consequential implementation risk and obtaining a distinguishing check without owner coaching or an enumerated defect in the model. This remains a capability to demonstrate; the sequential, process-local library example does not establish concurrency or database-risk discovery.

## Deliveries and evidence

Every Manager, including the root, returns the same delivery shape:

| Part | Required account |
| --- | --- |
| Binding | Task, attempt, source base and exact candidate. |
| Assessment | Each local Definition and how it relates to the realization. |
| Artifact changes | Actual harvested delta, registry changes and ownership claims. |
| Explanation | What changed, why, and which Definitions justify it. |
| Delegated results | Child deliveries and the aggregate candidate they were integrated into. |
| Evidence | Independent review verdicts, checks and exact candidate bindings. |
| Open work | Unresolved questions, conflicts, missing evidence and requests to other areas. |

The Orchestrator validates shape, scope, candidate bytes and lifecycle. A report cannot substitute claimed changes for the observed workspace delta.

Definition assessments are a small table, not a second semantic model. Each record holds the Definition, a disposition (`realized`, `constrains`, `vocabulary` or `unresolved`), the artifacts, the evidence and a reason. `realized` names the artifacts that realize the Definition; `constrains` names the other realization it constrains; `vocabulary` applies to declarations that need no standalone artifact. A positive disposition explains the observable realization or concrete use and why the evidence suffices; "done" or a file list is not enough. For behavior, the evidence names the tests or checks that would fail if the Definition were violated, and the reviewer judges whether they really discriminate. The Orchestrator checks the census, statuses, bindings and evidence references; the reviewer judges their substance. Every local Definition needs a disposition other than `unresolved`; an unresolved one prevents completion even when every file is registered.

The `vocabulary` disposition means a declaration needs no standalone artifact. It does not remove obligations its body or contracts impose on concrete uses; cite those uses and their assessments. The table shows what was examined; it does not prove that every natural-language obligation was understood. All artifact forms, including tests, are reviewed against their claimed model basis.

An unchanged Package can deliver a reviewed no-change result that still explains its realization and evidence. An old successful report is evidence for its old inputs, not current approval.

## Distinct review responsibilities

Each active named reviewer performs the required review responsibilities independently against the same frozen subject, including expectations before candidate evidence. Names and operational instructions describe perspectives, not permission to omit applicable obligations. Collect the complete roster; a generic individual pass cannot close another reviewer's finding. The [group and arbiter rules](manager-apply-execution.md#review-conflicts-and-arbitration) determine acceptance. All review conversations and arbitration start fresh; neither a repeated provider conversation nor the implementer's transcript supplies independence.

| Review | Question it must answer |
| --- | --- |
| Local delivery | Do this area's code, tests and documentation fulfill its Definitions and relevant connected obligations? Are effects on other areas accounted for? |
| Integration | Does the composed candidate satisfy the connected behavior and contracts across areas, such as a suspension preventing a new loan? At the root it also asks whether all areas follow the model's project-wide conventions uniformly. |
| Final evidence check | Deterministic, by the Orchestrator: does the exact aggregate have every required local and integration review bound to its current subject, passing required checks, conforming coverage and a complete Definition census, with no open blocking finding? |

The root's integration review is the semantic review of the whole system; no separate verifier Agent repeats it. Every local or integration review first records expectations from the bound model before seeing the candidate, then assesses that candidate and its evidence. These are two stages of the same review purpose, not another reviewer hierarchy. Project-wide conventions, such as how errors are reported, are model Rules with project scope, so the root's integration review can check them across areas. Where the configuration allows it, one reviewer session assesses several no-change deliveries, each with its own verdict bound to its subject. Batching never removes a required subject or reduces the [protected preparation floor](manager-apply-execution.md#one-ledger-with-protected-completion-capacity).

Each result is bound to its purpose and candidate. A local pass never replaces an integration review. Later stages can cite an earlier finding and its recorded resolution instead of repeating the discussion; a changed candidate, scope or assumption requires renewed assessment.

For the library demonstration, independent M1 and M2 expectations are frozen outside Agent write scope, so generated code and tests cannot redefine success together ([evaluation contract](library-example/evaluation-contract.md)). This evaluates the experiment; adopting projects do not maintain a second specification.

## Artifact registry and ownership

The registry lives in `.markitect/registry/`, one file per Manager that mirrors the Package folders: `.markitect/registry/lending/registry.yaml` for lending, `.markitect/registry/registry.yaml` for the root. The file determines the accountable Manager. Each record names its path, artifact form (`code`, `test`, `doc`, `build` or `config`), the Definitions it realizes and a short explanation:

```yaml
apiVersion: markitect.registry/v1
artifacts:
  - path: realization/internal/lending/borrow.go
    form: code
    realizes: [lending/borrow, lending/loan-policy]
    why: Admits a new loan only within the open-loan limit.
```

Only guarded publication writes registry files. An edit made outside Apply is drift: `check` validates it, and the next Apply's intake reassesses it. A registry revision is bound to the candidate tree. Review and check observations refer to that revision instead of being copied into mutable verdict fields.

Registry paths are independent of Package paths; lending can own source, test and documentation files in three directories. Other Managers can read or cite those files but cannot claim them. Managers propose registration through Orchestrator tools; the Orchestrator validates them and builds the registry delta. Editing the registry file directly is not a way around ownership.

Before writing a new path, a Manager reserves it through the Orchestrator, which checks the operation target, permitted creation areas, exact spelling and exclusive ownership. Model and configuration paths cannot be obtained through an artifact reservation. A shared file has one owner; others request changes through their common parent. A rename needs a new reservation and keeps provenance. A deletion removes the current registration only with a justified candidate deletion; history keeps the old registration. Moving responsibility between Packages is an explicit, recorded transfer reviewed with the affected delivery.

Coverage classifies every repository file as a canonical or Markitect input, a registered artifact or an explicitly ignored file ([DEC-006](../../concepts/register.md#dec-006-every-file-is-covered-or-explicitly-ignored)). Registrations need a valid owner and a resolvable model basis. Overlapping ownership, missing files, stale registrations and unexplained files are findings. A blanket ignore cannot turn missing realization into success. Ignore configuration is project-owned, fixed for the run and read-only during Apply; Managers cannot hide a file by adding an ignore. Coverage ignores never change the language loader's complete scan of Definitions.

### Markitect state

Markitect's own state stays out of the realization: the journal lives outside the repository, and the small control metadata in `.markitect/state/` is ignored by Git and excluded from source staleness ([journal](manager-apply-journal.md#physical-store-and-commit-contract)). Repository classification and ownership belong to the Orchestrator's coverage stage, which keeps the language compiler free of I/O.

### Removed and moved intent

Apply also removes realization whose intent is gone ([DEC-034](../../concepts/register.md#dec-034-removed-intent-removes-its-realization)). When a Definition disappears, every artifact whose only basis it was is orphaned. Its owner deletes it in the same Apply, or records a new justified basis that review accepts; an artifact that keeps other bases only loses the removed link. If the owning Package itself disappeared, its records pass to the nearest existing ancestor Manager, or to the root. Language v1 does not guess renames (section 9), so a moved or renamed Definition is a removal plus an addition: the link to the old identity becomes a blocking finding, and the owner re-registers the artifact against the new identity with review. When the move crosses Packages, the record moves to the new owner's registry file through a recorded transfer. Stale code from removed intent is not a matter of style; leaving it fails the reconciliation.

### Intake of unexplained files

Initially unexplained files go through read-only intake coordinated by the root: candidate registrations are justified against Definitions and independently reviewed before write tasks use them. Bounded inquiry follows direct parent-child edges to clarify responsibility within the already participating organization. It cannot create a new Manager or omit another Manager's assessment. Ambiguous responsibility remains a decision, not an automatic claim. Final publication requires conforming coverage of the aggregate candidate.

## Apply lifecycle

Logical Apply is the whole reconciliation below; guarded publication is its last step.

| Stage | Work and exit condition |
| --- | --- |
| Prepare | Read the model, configuration, registry and ignore policy from the committed state. Uncommitted changes to tracked files, or untracked files outside Git-ignored paths, stop the run before preparation. Acquire the repository snapshot and tool identities, compile v1 and bind the organization. Invalid model inputs stop the run. |
| Start root | Deliver the static instruction at the root and guarantee unchanged propagation to every local Manager with separately compiled context. Fix the artifact-only target and settle unresolved ownership through intake before write tasks begin. |
| Explore and plan | Every local Manager, including empty and intermediate Packages, reports an initial read-only assessment or submits a no-change result before any artifact writing ([T1–T3](manager-apply-core.md#manager-tasks)). Parents summarize their direct children. Local plans and dependency messages then progress without a global Plan barrier; the Orchestrator controls readiness. |
| Work and review | Local changes happen in an owned candidate workspace. The Orchestrator harvests the actual delta; a fresh reviewer assesses it. A failed review returns grounded rework; an unresolved model question pauses that task without editing the model. |
| Integrate upward | The parent's integration step receives the composed aggregate of its subtree ([T9](manager-apply-core.md#manager-tasks)) and checks interface and cross-area consistency. A fresh integration reviewer examines the aggregate and relevant child evidence. |
| Verify aggregate | At the root candidate: required project checks, then the Orchestrator's final evidence check over the whole Manager tree. The root's fresh integration review has already assessed the aggregate semantically. |
| Preflight | Recheck the captured repository, model, configuration, registry and ignore inputs. Drift makes the run stale; earlier checks cannot authorize changed bytes. |
| Publish | If the configuration requires the owner's approval for an affected Package, the Orchestrator raises an approval question with the result summary, and the run waits for the answer. The guarded writer then applies the accepted candidate and registry revision to the working tree. Apply never commits, merges, releases or deploys. |
| Read back | Confirm actual paths, bytes, modes and registry against the verified candidate. A partial write or failed read-back is not a successful Apply. |

Full reconciliation of the selected local model is the default; it does not claim coverage of intent outside the model. Do not implement incremental reuse first; [computed scope](#toward-computed-scope) follows later. The graph and registry can explain possible impact; unknown semantic impact broadens examination.

The Orchestrator keeps one accepted aggregate per run and composes each accepted delivery onto it by replaying its owned delta ([D2](manager-apply-core.md#deliveries-and-composition)); every step starts from the current aggregate ([D3](manager-apply-core.md#deliveries-and-composition)). A failed composition condition is an Orchestrator error that stops the run, not a merge. A parent never writes child-owned paths; a correction to child content is an issue addressed to that child ([D5](manager-apply-core.md#deliveries-and-composition)). Parent-owned integration artifacts can be changed by the parent. [D4](manager-apply-core.md#deliveries-and-composition) decides which reviews composition renews. No last-writer-wins merge or silent reassignment resolves a conflict.

The owner's commit of the published result is the acceptance, exactly as a committed model revision is the accepted specification ([DEC-035](../../concepts/register.md#dec-035-committed-model-in-owner-commit-accepts-the-result)). The run report proposes a commit message with the trailers `Markitect-Run` and `Markitect-Model`. After a partial publication or a result the owner does not accept, the owner commits or reverts the working tree before the next run.

The rejection allowance is finite, and after repeated rejections the parent assesses the plan; the [execution contract](manager-apply-execution.md#admission-amendments-and-finite-correction) owns counting, budgets and exhaustion. A parent cannot turn an unresolved child delivery into a complete one by summarizing it.

The [execution contract](manager-apply-execution.md#one-run-state-machine) owns the single run state machine and its [terminal outcomes](manager-apply-execution.md#terminal-outcomes-and-displayed-status). History survives interruption: reopening a run first reconciles recorded invocations, child lifecycle and publication outcomes. Completed provider work is not replayed automatically, and unknown side effects stay visible.

## Reports and execution history

One run history serves two views: concise upward reports for coordination and a drill-down record for investigation. Reports reference evidence; they do not replace it. A parent can request underlying sources, candidate differences and conversations.

At minimum the history records run, task, attempt and Agent identities, causal links, effective role configuration, model and candidate bindings, prepared context, actual rendered prompts and custom instructions, sent requests, returned messages and reports, context and tool reads, delegation, reservations, candidate deltas, review submissions and rejections, checks, integration, publication and read-back. Original events are kept; corrections are new events. Large payloads are content-addressed blobs. Prepared, sent and confirmed or unknown delivery states stay distinct, so a built template never counts as a received request. Opaque provider behavior is a visibility limit, not a transcript Markitect may invent.

Every candidate transition identifies its prior and resulting tree and a retrievable difference. Records distinguish a proposed change, an Agent claim, an Orchestrator-observed change, a reviewed aggregate and an applied result. Rejected candidates stay inspectable. Parallel causal branches are not flattened into a false order by timestamps.

The normal user report presents the result, model basis, evidence, limitations and decisions needing attention. A small coordination account from the same records shows local work and no-change outcomes, cross-area requests, invocations, known usage and elapsed time. It separates a Manager's own effort from descendant work and waiting, and unknown usage stays unknown. It always reports the same measures: Agent steps, sessions and tokens per role, review submissions and rejections per purpose, findings by kind and severity, the share of no-change deliveries, owner interventions and active execution time. These feed the evaluation of [DEC-009](../../concepts/register.md#dec-009-the-central-evaluation-question) and [PRM-005](../../concepts/register.md#promises). The account helps judge whether one Manager per Package works in practice; it adds no scores, thresholds or automatic reorganization.

A failure to record required work or publication events durably stops further dispatch or publication. Recorded inputs and outputs support inspection; they do not guarantee replay, semantic correctness or human acceptance. The first inspection surface is a run summary, the Manager task tree, an event timeline and per-step access to input, report, candidate difference and evidence. No cockpit is needed for the first Apply.

## Library walkthrough

The [library delegation walkthrough](manager-apply-library-flow.md) traces the root, lending, a rejected six-loan candidate, fresh review, rework, reminders and integration against M1. The [library M1](library-example/README.md) has the Packages library (shared scope and rules), catalog, members, lending and lending/reminders, with explicit connections from membership and availability to lending. The same lifecycle covers drift repair: if the existing lending code, its test and the usage guide allow a sixth loan while the model is unchanged, Apply realigns all three against the five-loan rule without a model edit.

## Design checks and failure handling

These counterexamples check the contracts; each becomes a test in the package that owns it.

| Situation | Required behavior |
| --- | --- |
| Artifacts drift while the model is unchanged | Apply assesses and reconciles them; no artificial model change is needed. |
| A binding rule has no needed context link | If noticed, report the modeling gap. Neither edit the model nor treat the missing link as cancelling the rule. |
| A local statement seems to require another area's behavior | Manager and reviewer raise a source-grounded question through the common parent. An unresolved required assessment blocks completion. |
| Two Managers claim the same file | Reject the overlapping claim and coordinate one owner. Output order never picks a winner. |
| A worker changes a canonical Definition during Apply | Reject the candidate as outside the operation target. |
| A `vocabulary` disposition hides a required feature | Independent review rejects the unsupported assessment. |
| A Kind needs no artifact but constrains its uses | Assess the concrete uses and their realization. |
| Every file is registered, but a sixth loan is still allowed | Coverage can pass; semantic review and domain checks must catch the behavior. |
| Every file is owned, but a Definition has no justified disposition | Refuse completion. |
| Child review passed, but composition breaks the shared contract | Integration review fails; the owning area repairs and the new candidate is revalidated. |
| A local pass is offered as integration or final evidence | Reject the substituted review purpose. |
| A parent needs a child's file to judge integration | Supply a recorded targeted read. There is no file-blindness rule. |
| An external edit changes the checkout or model during work | Preflight reports stale and preserves the candidates. |
| Review cannot converge or a model contradiction remains | Finish Incomplete within the rejection and assignment limits, with the open question and retained evidence. |
| A Definition was removed, but its artifacts remain | Their owner deletes them or records a new basis that review accepts; otherwise the run cannot complete. |
| A Definition moved to another Package | The link to the old identity is a blocking finding until the owner re-registers the artifact; a move across Packages transfers the record. |
| The model or another file has uncommitted changes when Apply starts | Refuse before preparation. |
| Configuration or custom instructions state a project rule | The reviewer records a finding; the rule moves into the model. |
| Publication writes some files and then fails | The run reconciles until the actual writes are known. A known partial result finishes Failed with its exact paths. |
| An Agent reports success without a recorded delta or evidence | The Orchestrator refuses completion. |
| A required event cannot be stored durably | Stop before further dispatch or publication. |
| A rendered context exceeds the size limit | Stop preparation with a finding to split the Package. |
| Composed deliveries touch the same file | Orchestrator error; ownership is broken and the run stops. |
| A consumer reads another area's files or data directly | The integration review records a finding; the consumer must use the producer's API. |
| A provider call fails before anything was sent | Retry automatically within the configured limit, each attempt charged; otherwise reconcile. |

## Toward computed scope

Computed scope, the vision's first mechanism, follows once the two-Apply cycle works. It derives the affected Managers from the model difference per Definition, through the direct references and the registry links. Only those Managers receive tasks, and unknown impact still broadens the scope. Managers outside it get no task, and their last accepted realization stands. Because declared references cannot show every code dependency ([ENH-001](../../concepts/register.md#enh-001-observed-code-dependencies-widen-impact)), a full reconciliation stays available and advisable from time to time. The data this needs is recorded from the first run on: Definition digests per run, registry links and the reference graph.

## Relation to current source and delivery

The [rebuild plan](manager-apply-rebuild-plan.md#source-replacement-map) decides which current mechanisms are reused, rewritten or deleted; the existing Reconcile operation already asks for complete responsibility against unchanged intent and reasoned no-change results. The [delivery plan](../../implementation-plan.md) owns packages and order, and the [acceptance gates](manager-apply-implementation-plan.md) own what each step must prove. Further commands, team strategies, concurrency and incremental reuse wait until the two-Apply cycle is demonstrated. Government remains a later, separate authority over model changes ([DEC-004](../../concepts/register.md#dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis)).
