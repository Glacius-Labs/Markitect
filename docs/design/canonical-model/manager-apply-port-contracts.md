# Port contracts for the first reliable Apply

This contract owns the Go interfaces between the Apply core and its adapters, the values they exchange and their behavioral rules. The [Go architecture](manager-apply-go-architecture.md) owns package boundaries; the [library walkthrough](manager-apply-library-flow.md) shows these exchanges in sequence. Provider wire formats belong to the adapters.

## Values, identities and ownership

| Value | Owner and required content |
| --- | --- |
| Fixed tree and content reference | `snapshot`: explicit paths, bytes or content identities and relevant file modes. No repository acquisition or workspace identity. |
| Definition reference | `model`: complete resolved v1 identity with source provenance. Runtime tasks add no second model identity scheme. |
| Manager identity | `organization`: one ordinary identity per local Package, including the root. Imported vocabulary creates no Managers. |
| Input binding | `run`: model closure and lock, Package inventory, configuration and instructions, ignore rules, initial registry, repository census and artifact source, plus the selected executor and check profiles. `apply` builds it from captured inputs. A model digest alone is insufficient. |
| Candidate reference | `realization`: an immutable artifact tree with its matching registry revision and base provenance. Not a workspace path. |
| Task and attempt | `run`: task ID, parent, Manager, operation, complete responsibility, outcome, model basis, input binding, base candidate and budget view. Rework adds an attempt; it never invents a new responsibility. |
| Assignment and continuation | `run`: one owner request with its caps and linked fixed-basis runs. Continuation runs get new task and session identities with provenance links; retained work inherits no approval. |
| Context package | `organization` renders it, including the mandate section; `run` binds it to the task. The reviewer assesses the mandate's authority under its own read-only grant. |
| Actor, session and step | `run`: role and scope, logical session ID and unique dispatch step ID, assigned before any effect. Provider handles stay adapter data. |
| Review subject | `run`: input binding, exact candidate with registry, task scope and purpose (local or integration). |
| Accepted delivery | `run`: Host-issued ID, exact subject, delivered scope, Definition assessments, child delivery IDs, review and check evidence, report. Agents cannot create one. |
| Task dependency | `run`: the producer task and the accepted delivery selected for composition. No mutable "latest" alias. |

Markitect's state in `.markitect/state/` is excluded from source staleness; adding or deleting any other repository file always changes the captured census.

```go
// Package run. Referenced domain values belong to their defining packages.
type TaskSpec struct {
	ID             TaskID
	Assignment     AssignmentID
	Parent         TaskID
	Predecessor    *TaskRef
	Manager        organization.ManagerID
	Attempt        uint32
	Operation      Operation
	Responsibility Responsibility
	Outcome        string
	ModelBasis     []model.DefinitionRef
	Inputs         InputBinding
	Context        organization.ContextRef
	Base           realization.CandidateRef
	Dependencies   []TaskDependency
	Budget         BudgetView
}

type WorkProposal struct {
	Summary     string
	Assessments []realization.DefinitionAssessment
	Registry    []RegistryProposalID
	Children    []DeliveryID
	Questions   []QuestionRef
}

type ReviewProposal struct {
	Subject      ReviewSubject
	Verdict      ReviewVerdict
	Findings     []Finding
	FindingMoves []FindingStateChange
	Evidence     []EvidenceRef
}
```

`BudgetView` is a read-only snapshot of the remaining caps for planning. Admission always uses the current ledger, never that snapshot ([execution contract](manager-apply-execution.md)).

The sender of an adjacent message is the bound Manager session; it cannot choose its own authority, base, attempt or source binding. The Host permits only its direct parent or children and preserves each recipient's complete responsibility, resolved constraints and unresolved obligations. A message cannot create participation, replace the operation or narrow reconciliation to a cited subset of Definitions.

Work proposals are explanations, not trusted diffs. Definition assessments cover every local Definition with a disposition, grounds and evidence references; a `vocabulary` disposition must show where concrete uses meet their obligations. Registry proposals refer to Host-authorized path claims. A no-change proposal needs the same assessment as a changed candidate.

A `Finding` has a Host-recorded ID, a severity (blocking or advisory), the affected responsibility, its model and input basis, the requested evidence and the originating reviewer; a candidate reference is optional for concerns raised before a candidate exists. A `FindingStateChange` moves an existing finding to resolved (with current evidence) or not applicable (with grounds), as defined in the [execution contract](manager-apply-execution.md#evidence-need-survives-budget-pressure). Only a bound independent reviewer can issue one. Verdicts distinguish pass, changes requested, needs decision and failed assessment. The Host derives the open findings from task, child and predecessor history and validates provenance and bindings; whether evidence suffices remains the reviewer's judgment.

A dependent task starts only from an accepted producer delivery composed into its base ([execution contract](manager-apply-execution.md#accepted-deliveries-and-dependent-work)). Step and workspace records bind what was actually supplied: prompts, context, composed deliveries, continued-session history and later replies.

`QuestionRef` binds identity and version. The application binds owner answers to that exact proposal; an Agent cannot assert owner authority. A context-only answer may continue the same run; changed intent ends the run Incomplete and needs a committed model edit and a continuation ([journal](manager-apply-journal.md#questions-answers-and-model-changes)).

The illustrative `TaskSpec.Outcome` above is the static Host operation instruction, not parent-authored prose. Its model basis and complete scope are compiler-derived. Additional work messages remain separate and cannot redefine that mandate. Exact responsibility and prerequisite records must follow the remaining [process details](manager-apply-concept.md#remaining-process-detail-and-technical-boundary); existing provider Session contracts are not a Manager scheduler.

The confirmed process uses normal deliveries with limited scope while their Manager remains open. The technical representation must bind Host acceptance and the consumer's composed basis, separately from a reviewer pass. Do not invent a parallel receipt lifecycle or a budget reset for each limited delivery. Successful deliveries consume shared execution capacity but neither use nor reset the rejection counter, whose identity is fixed per run, Manager and review purpose by the [execution contract](manager-apply-execution.md#admission-amendments-and-finite-correction). Exact counting, binding and recovery need implementation evidence.

## Bounded child reports

The owner selected this target message structure; it is not an implemented wire schema or a second delivery ledger. Use the existing adjacent-message channel and journal. The Host binds sender, recipient, run, Manager and exact step; a child cannot assert those identities or issue an accepted delivery.

| Part | Required meaning |
| --- | --- |
| Identity and basis | Report ID/version, exact originating step and workspace/candidate binding, and any accepted delivery reference. A later report identifies what it supersedes. |
| Responsibility and status | Covered scope and initial assessment, progress, limited delivery or completion proposal. Remaining responsibility stays explicit; only the Host can confirm completion. |
| Result and interfaces | Short result, changed interface references and implications for the direct parent. No descendant transcript dump. |
| Prerequisites and replies | Needed results, exact producer/delivery references when known, outstanding questions and replies required to proceed. |
| Findings and uncertainty | Open finding references, unresolved assumptions and observation limits. Absence from a summary never resolves an existing finding. |
| Evidence and detail | Exact immutable evidence/detail references with the scope each supports. An Agent's report remains a proposal until required review and Host acceptance. |

The Host enforces finite report and assembled-input byte limits from a versioned policy. Exact field names and numeric defaults are producer work, not additions to today's execution YAML. Keep the complete direct-child census and outstanding obligations in the existing journal; only bounded views enter a turn. If a report or parent input exceeds its limit, split it visibly into version-bound parts with stable ordering, an inspectable index and an explicit remainder. Existing message and step observations record which parts were supplied; a changed report version requires current coverage. There is no separate paging ledger, and a delivery acknowledgement does not prove semantic understanding. Required findings and prerequisites cannot be silently truncated or disappear behind a successful summary. If required material cannot be supplied within the finite limits, retain incomplete work and report that boundary.

Targeted detail retrieval uses evidence references and existing access rules, never direct sibling communication or a provider-selected participation list. Reading every descendant file is not the default. Summarization may judge significance but cannot waive a child, an outstanding question or a mandatory review. The first Apply still obtains a fresh initial assessment from every Manager; report paging is not assessment reuse.

## Application boundary and source acquisition

CLI and MCP call a concrete `apply.Service.Apply(ctx, request)` and receive a result plus an ordinary error. The request names the repository and either starts a new assignment or continues an open one by its ID; a repository with an unfinished run refuses a new Apply until that run is resumed to its end or stopped ([execution contract](manager-apply-execution.md#assignments-and-runs)). Apply depends on one consumer-owned acquisition port:

```go
// Package apply.
type Source interface {
	Capture(context.Context, CaptureRequest) (CapturedInputs, error)
}
```

`CaptureRequest` names the repository, canonical root, explicit dependencies and configuration. The model, configuration, registry and ignore inputs are read from the committed state; uncommitted changes to tracked files, or untracked files outside Git-ignored paths, refuse the capture before preparation. `CapturedInputs` supplies fixed bytes and trees, the repository census, configuration, registry and ignore inputs and an opaque publication guard, with the identities needed to reproduce them. It never guesses dependencies from prose.

Apply compiles those bytes with `model`, derives organization and context and validates initial file accounting. Unknown files need a classification before they can change. Model, configuration and independent expectations stay outside Agent write authority. Invalid inputs stop work before dispatch, and the bound inputs are stored durably before any session starts. `apply.Control` supplies the controller lock and journal ([journal](manager-apply-journal.md#physical-store-and-commit-contract)); CLI and MCP never acquire execution authority themselves.

## Session and communication port

```go
// Package run.
type Sessions interface {
	Open(context.Context, SessionSpec, SessionIO) (SessionObservation, error)
	Step(context.Context, StepRequest, SessionIO) (StepReceipt, error)
	Inspect(context.Context, SessionQuery) (SessionObservation, error)
	Stop(context.Context, SessionID) (SessionObservation, error)
}

type SessionIO interface {
	Observe(context.Context, SessionEvent) error
	Call(context.Context, ToolCall) (ToolResult, error)
}
```

| Record | Required fields and meaning |
| --- | --- |
| `SessionSpec` | Host execution-session ID, stable Manager identity, known conversation identity or explicit fresh-conversation handoff, role, executor profile and runtime image, fixed workspace access/authority ceiling and supplied history references. Resume capability is established, not inferred. Review sessions start fresh. |
| `StepRequest` | Unique step ID, execution session and conversation association, task and exact admitted workspace version, consumed accepted deliveries, own starting changes, subject binding, exact prompt/context/instruction references, additional messages and permitted effects. Recorded before dispatch. |
| `StepReceipt` | Step and session identity, observed status, proposed result or yield, output references, usage when observable and observation limits. Yield, proposed completion, needs decision, failure and uncertain interruption are distinct; none is an accepted delivery. |
| `SessionQuery` | Known session ID and, where relevant, step ID. Unavailable information stays unknown. |
| `SessionEvent` | Session and step, sequence and correlation, kind and exact observed payload, including prepared outbound payloads, transport observations, incoming content and tool exchanges. |
| `ToolCall` / `ToolOutcome` | Correlation ID and a closed typed request for adjacent messaging, yielding, targeted reads, path claims, registry proposals, questions, allowed checks or result submission. `Call` records it, the Host answers it and the answer is recorded before the adapter returns it unchanged: `ok` with a result, or `refused` with a reason, which the adapter reports as a tool error. The sender is the session whose container channel carried the call. |

Every session runs in its own container through the session helper `markitect-agentd`, which serves the Host tools to the provider CLI over MCP and carries everything over one stdio connection ([rebuild plan](manager-apply-rebuild-plan.md#container-runtime)). The Host implements `SessionIO`. The adapter calls `Observe` with the actual outgoing content and waits for durable acknowledgement before sending it. Hidden provider context and unobservable activity are reported as limits, never reconstructed from templates. An executor that cannot support this protocol and its input observation is not selectable.

The records above describe the selected target; the implemented Go API is `run.Sessions` and `run.SessionIO` in source. In the first strategy one execution session is one container serving exactly one step, so the session plan's workspace and candidate are the step's. The plan also names a provider-home volume that survives the container and, when continuing, the provider conversation to resume; a resumed session binds only under that identity, and reviewers never resume. The admissions hand the adapter the exact recorded profile, tool, input and contract bytes. A source-only continuation claim is not a physical capability.

Conversation continuity does not permit concurrent mutation of its workspace. The Host permits a basis advance only between observed quiescent steps and records the [working-basis transition](manager-apply-concept.md#working-bases-and-conversation-continuity). Every action is checked against the active step and version. A late request from an older step cannot write or create new current evidence; a repeated completed request returns only its old receipt. Independent review uses its frozen subject rather than a Manager's advancing workspace.

`Call` validates role, active step/version, permitted effects, task state and every adjacent parent-child communication edge. Before the complete Explore census, all artifact mounts are read-only and the Host denies artifact-mutating tools and workspace effects; role alone cannot grant them. Read-only analysis and proposed classifications remain distinct from mutation. Initial operation propagation is a Host obligation independent of tool calls or Agent output. A Manager can submit a question, dependency response or result and yield; a callback never runs a session recursively or waits for a child workflow. Reviewers can read, request allowed checks and report; they cannot claim paths, change ownership or control Manager participation.

The tools form one closed catalogue; each role sees only its own. DEC-039 removes the former `plan` and `delegate` controls. The adjacent-message schema must follow the [fixed process](manager-apply-concept.md#tasks-and-delegation) before implementation; the table does not freeze that API:

| Tool | Roles | Effect |
| --- | --- | --- |
| Adjacent message (schema pending) | Manager | Sends a bounded question, response or result to a direct parent or child. Host checks each hop and records pending replies; this cannot create a narrower operation or omit a Manager. |
| `read(ref)` | Manager, reviewer | Returns a Definition, artifact, delivery report or transcript by identity and records what was supplied. |
| `reserve_path(path, form)` | Manager | Reserves a new artifact path within the creation areas and exclusive ownership. |
| `register(path, realizes, why)` | Manager | Proposes a registry change for an owned, reserved or intake-assigned path. |
| `assign_owner(path, manager, reason)` | Manager | As the common parent, decides the owner of an artifact that several descendants' Definitions require, assigns an unexplained file during intake, or transfers a record between its descendants. |
| `ask(question)` | Manager | Raises a question or a sibling request to the parent. |
| `answer(question, response)` | Manager | Answers a child's question within the mandate, or routes it to a sibling or upward; owner answers arrive only through the application. |
| `request_check(check)` | Manager, reviewer | Runs an allowed project check on the current candidate and returns its receipt. |
| `submit_work(proposal)` | Manager | Submits a work proposal, a no-change delivery or a yield while children or answers are pending. |
| `record_expectations(cases)` | Reviewer | Records the expected behavior and discriminating cases derived from the model before the candidate is supplied. |
| `submit_review(proposal)` | Reviewer | Submits a review verdict with findings and evidence. |

Tool deduplication binds the actual execution session, channel, step and call identity. An exact repeated request with identical content returns its original recorded reply; changed content under that identity is an error. A reused provider call ID in another step or channel is not evidence of the same invocation and must not inherit its receipt or authorize replay of unresolved earlier work. An uncertain original call stays uncertain until reconciled. Duplicate observations create no tasks or attempts. Cancellation propagates through `context.Context`. A timeout or error after possible dispatch returns the known receipt with an uncertain state; inspect the known session and step before retrying, and never resend work just because a reply was lost. `Stop` reports observed termination, not only a cancellation request. An uncertain writer blocks workspace handoff. A `SessionIO` callback error prevents further dispatch; external activity that may already have started stays uncertain until reconciled.

## Candidate workspace port

Workspace materialization must preserve own unaccepted changes across an explicitly admitted basis advance or refuse the transition with that work retained. The existing pure `ComposeCandidates` requires an exact common base and disjoint effects; it is not an arbitrary-base rebase, a live workspace switch or a delivery-acceptance API. A producer must establish a common base and complete before/after facts or return the conflict for owned correction. The following port shape does not itself establish a safe transition, physical quiescence or restart behavior.

```go
// Package run.
type Workspaces interface {
	Open(context.Context, WorkspaceSpec) (WorkspaceObservation, error)
	Freeze(context.Context, FreezeRequest) (Harvest, error)
	Materialize(context.Context, MaterializeRequest) (WorkspaceObservation, error)
	Inspect(context.Context, WorkspaceID) (WorkspaceObservation, error)
	Close(context.Context, WorkspaceID) (WorkspaceObservation, error)
}
```

| Record | Required fields and meaning |
| --- | --- |
| `WorkspaceSpec` | Host workspace ID, task and attempt, exact base candidate, protected input binding and role access. Writes are granted to owned paths and reserved new paths. |
| `WorkspaceObservation` | ID, materialized base, access, lifecycle state and recovery information, also after partial preparation. |
| `FreezeRequest` | Workspace ID, expected base and current registry ledger, after the writing step ended. An unknown active writer prevents freezing. |
| `Harvest` | The full observed tree and delta, protected-input comparison, ownership violations and, when valid, the immutable candidate with its registry. A proposal's file list never limits observation. |
| `MaterializeRequest` | A new workspace ID and a fixed `realization.CompositionPlan`, expressed through snapshot and realization values. |

The Host owns claims and decisions; the adapter observes files. New paths need a reservation before writing, and a later claim never authorizes an earlier unauthorized write. Freeze observes all relevant paths, including deletions and protected model and configuration inputs; read-only review access is checked the same way. Workspaces are volumes of the container runtime; the adapter materializes, observes and harvests them with Git inside a helper container, so observation follows Linux file semantics. A session container mounts only its own workspace. The checks do not claim protection against processes outside the container runtime.

A frozen candidate is immutable; checks and reviews use it, never a mutable path with the same name. Before another writing attempt, the workspace must match its expected start or be rematerialized. Rejected candidates are stored with their evidence. Composition rules are pure functions in `realization` that compare exact bases, owned changes and registry changes and return a fixed plan. A disjoint union requires a proven common base and disjoint owned deltas; sequential writing alone does not establish either. An overlap or unproven base composition is refused and returned for owned resolution, never silently rebased. `Materialize` applies only a validated plan and reports what it observed; it never merges competing semantics or picks an owner.

## Check execution port

```go
// Package run; also used by final Apply verification.
type Checks interface {
	Run(context.Context, CheckRequest) (CheckReceipt, error)
}
```

`CheckRequest` binds a Host check ID, the exact review subject and an allowed check from fixed configuration, with its runtime image, executable, arguments, environment, limits and tool identity. A reviewer may request an allowed check; a missing artifact test becomes a finding for its owner.

`CheckReceipt` carries the actual exit or termination state, output references, executable identity, candidate binding and observed drift or limits. A nonzero exit is a check failure; a startup or observation failure is reported separately. Cancellation never turns an unknown process into a pass. Checks run in a container from the check profile's image, on a disposable copy of the fixed candidate. An Agent saying "tests pass" is not a check receipt.

## Durable journal port

```go
// Package run; apply uses the same journal for its lifecycle.
type Journal interface {
	Put(context.Context, Payload) (BlobRef, error)
	ReadBlob(context.Context, BlobRef) (Payload, error)
	Append(context.Context, AppendBatch) (AppendReceipt, error)
	Read(context.Context, EventQuery) (EventPage, error)
}
```

Payloads are immutable content-addressed bytes whose references keep media and encoding information; reads verify identity. A batch names a unique batch ID, the expected revision of the repository's stream and typed events, each bound to its assignment, run, task, actor, session, step, attempt, candidate and causal predecessors. Referenced payloads are durable before acknowledgement. Append is one logical commit under one coordinated writer; identical repeats return the original receipt, different bytes are rejected, and a stale revision stops dispatch. The [journal contract](manager-apply-journal.md) owns the physical store. A required journaling failure prevents further work but never erases an effect that may already have happened.

## Guarded publication port

```go
// Package apply.
type Publisher interface {
	Preflight(context.Context, PublicationRequest) (PublicationPermit, error)
	Publish(context.Context, PublicationPermit) (PublicationReceipt, error)
	Inspect(context.Context, PublicationID) (PublicationReceipt, error)
}
```

`PublicationRequest` holds a publication ID, the original acquisition guard and input binding, the exact verified candidate with its registry, the final evidence references and the artifact-only target. It also carries the controller check, a run-scoped Host authorization and the run and epoch binding; a live controller alone cannot authorize an obsolete run, and a finished run's unused permits are void. Preflight checks readiness and source state; its permit is bound to that request.

Publish writes into the working tree and never commits, merges or pushes; the owner's commit accepts the result. It revalidates the source guard at the write boundary, including additions, deletions and input changes, writes only the verified artifact and registry change and reads the result back. Its receipt distinguishes applied, verified no change, stale, partial, unknown and failed, with the completed paths, observed tree and registry identities and observation limits, also when an error is returned; callers keep such receipts. Multi-file publication is not atomic. Partial writes stay visible and recorded; the owner commits or reverts them before a later run captures the state. Even Applied is machine evidence, not human acceptance.

## Host lifecycle and completion

The Host guarantees the same operation reaches every local Manager and runs one writer at a time with recorded ready and waiting tasks. The ready-work policy is fixed by the product, not supplied by an Agent. Managers act only on messages; no silent skip or pass-through startup exception is allowed. A task moves through local work, checks and review; a rejection starts the next bounded submission. A Manager with children also composes their deliveries and obtains integration review before the Host accepts its subtree. The [execution contract](manager-apply-execution.md#one-run-state-machine) owns run states, outcomes and admission; this port defines no other lifecycle. Typed results keep the current state, any terminal outcome, reasons and observation limits even when a Go error is returned.

Before any artifact write, the Host requires the initial read-only Explore assessment from every Manager. Local planning thereafter has no global barrier. Dispatch follows [recorded readiness](manager-apply-concept.md#ready-work-and-parent-guidance); dependency waiting cannot retain an active write grant. Separate parent hints preserve the operation, compiled context and authority. These are admission requirements within the existing lifecycle, not additional run states.

The root is ordinary in this protocol. `apply` adds the final evidence check and publication after the root delivery. Evidence stays bound to its subject; changed content or assumptions require current assessment. Source drift prevents publication even when every Agent reported success.

## Focused acceptance cases

The [walkthrough's refusal cases](manager-apply-library-flow.md#concrete-refusal-cases) are the first controlled-adapter checks. Add durable recording before dispatch, unknown dispatch and publication recovery, registry and tree mismatch, protected-input drift, exact duplicates and immutable review subjects. Then prove session start, Host calls, yielding, resumed input, fresh review and termination with the selected real executor, separately from the two real Applies.

Interfaces belong to their consumers, and adapters return concrete types ([Go interface guidance](https://go.dev/wiki/CodeReviewComments#interfaces)). Effectful calls take `context.Context` first ([Go context](https://go.dev/blog/context)). Pure model, organization and realization functions need no port just to be testable.
