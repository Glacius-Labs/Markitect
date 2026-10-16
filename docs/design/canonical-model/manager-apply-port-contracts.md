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

The journal and scratch storage are excluded from source staleness only through the fixed storage policy; adding or deleting an ordinary repository file always changes the captured census.

```go
// Package run. Referenced domain values belong to their defining packages.
type DelegateProposal struct {
	To          organization.ManagerID
	Outcome     string
	ModelBasis  []model.DefinitionRef
	Constraints []string
}

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
	Mandate        organization.MandateRef
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

The sender of a delegation is the bound session; it cannot choose its own authority, base, attempt or source binding. The Host allows only a direct child and keeps the child's full Package responsibility, inherited restrictions and unresolved obligations. A cited subset of Definitions never narrows reconciliation.

Work proposals are explanations, not trusted diffs. Definition assessments cover every local Definition with a disposition, grounds and evidence references; vocabulary-only must show where concrete uses meet their obligations. Registry proposals refer to Host-authorized path claims. A no-change proposal needs the same assessment as a changed candidate.

A `Finding` has a Host-recorded ID, a severity (blocking or advisory), the affected responsibility, its model and input basis, the requested evidence and the originating reviewer; a candidate reference is optional for concerns raised before a candidate exists. A `FindingStateChange` moves an existing finding to resolved (with current evidence) or not applicable (with grounds), as defined in the [execution contract](manager-apply-execution.md#evidence-need-survives-budget-pressure). Only a bound independent reviewer can issue one. Verdicts distinguish pass, changes requested, needs decision and failed assessment. The Host derives the open findings from task, child and predecessor history and validates provenance and bindings; whether evidence suffices remains the reviewer's judgment.

A dependent task starts only from an accepted producer delivery composed into its base ([execution contract](manager-apply-execution.md#accepted-deliveries-and-dependent-work)). Step and workspace records bind what was actually supplied: prompts, context, composed deliveries, continued-session history and later replies.

`QuestionRef` binds identity and version. The application binds owner answers to that exact proposal; an Agent cannot assert owner authority. A context-only answer may continue the same run; changed intent ends the run Incomplete and needs a committed model edit and a continuation ([journal](manager-apply-journal.md#questions-answers-and-model-changes)).

## Application boundary and source acquisition

CLI and MCP call a concrete `apply.Service.Apply(ctx, request)` and receive a result plus an ordinary error. The request names the repository and either starts a new assignment or continues an open one by its ID; a repository with an unfinished run accepts only the continuation of that run's assignment ([execution contract](manager-apply-execution.md#assignments-and-runs)). Apply depends on one consumer-owned acquisition port:

```go
// Package apply.
type Source interface {
	Capture(context.Context, CaptureRequest) (CapturedInputs, error)
}
```

`CaptureRequest` names the repository, canonical root, explicit dependencies and configuration. The model, configuration, registry and ignore inputs are read from the committed state; uncommitted changes there, or anywhere in the working tree outside ignored paths, refuse the capture before preparation. `CapturedInputs` supplies fixed bytes and trees, the repository census, configuration, registry and ignore inputs and an opaque publication guard, with the identities needed to reproduce them. It never guesses dependencies from prose.

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
	Call(context.Context, ToolCall) (ToolReply, error)
}
```

| Record | Required fields and meaning |
| --- | --- |
| `SessionSpec` | Host session ID, role, Manager, task and attempt, executor profile identity, workspace access and explicitly supplied history. Review sessions start fresh. |
| `StepRequest` | Unique step ID, session ID, task and subject binding, exact prompt, context and instruction payload references, additional messages and tool grants. Recorded before dispatch. |
| `StepReceipt` | Step and session identity, observed status, proposed result or yield, output references, usage when observable and observation limits. Yield, proposed completion, needs decision, failure and uncertain interruption are distinct; none is an accepted delivery. |
| `SessionQuery` | Known session ID and, where relevant, step ID. Unavailable information stays unknown. |
| `SessionEvent` | Session and step, sequence and correlation, kind and exact observed payload, including prepared outbound payloads, transport observations, incoming content and tool exchanges. |
| `ToolCall` / `ToolReply` | Correlation ID and a closed typed request or result for delegation, yielding, targeted reads, path claims, registry proposals, questions, allowed checks or result submission. The sender comes from the session channel. |

The Host implements `SessionIO`. The adapter calls `Observe` with the actual outgoing content and waits for durable acknowledgement before sending it. Hidden provider context and unobservable activity are reported as limits, never reconstructed from templates. An executor that cannot support this protocol and its input observation is not selectable.

`Call` validates role and task state. A delegate call records a child task and returns its ID; the parent then yields, and the child is scheduled outside the callback. A callback never runs a session recursively or waits for a child workflow. Reviewer tools can read, request allowed checks and report; they cannot claim paths, change ownership or delegate.

The tools form one closed catalogue; each role sees only its own:

| Tool | Roles | Effect |
| --- | --- | --- |
| `delegate(to, outcome, basis, constraints)` | Manager | Records a task for a direct child and returns its ID. |
| `read(ref)` | Manager, reviewer | Returns a Definition, artifact, delivery report or transcript by identity and records what was supplied. |
| `reserve_path(path, form)` | Manager | Reserves a new artifact path within the creation areas and exclusive ownership. |
| `register(path, realizes, why)` | Manager | Proposes a registry change for an owned or reserved path. |
| `ask(question)` | Manager | Raises a question or a sibling request to the parent, which may route it to the owner. |
| `request_check(check)` | Manager, reviewer | Runs an allowed project check on the current candidate and returns its receipt. |
| `submit_work(proposal)` | Manager | Submits a work proposal, a no-change delivery or a yield while children or answers are pending. |
| `submit_review(proposal)` | Reviewer | Submits a review verdict with findings and evidence. |

The Host deduplicates a repeated correlation ID with identical content and returns the recorded reply; different content under the same ID is an error. Duplicate observations create no tasks or attempts. Cancellation propagates through `context.Context`. A timeout or error after possible dispatch returns the known receipt with an uncertain state; inspect the known session and step before retrying, and never resend work just because a reply was lost. `Stop` reports observed termination, not only a cancellation request. An uncertain writer blocks workspace handoff. A `SessionIO` callback error prevents further dispatch; external activity that may already have started stays uncertain until reconciled.

## Candidate workspace port

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

The Host owns claims and decisions; the adapter observes files. New paths need a reservation before writing, and a later claim never authorizes an earlier unauthorized write. Freeze observes all relevant paths, including deletions and protected model and configuration inputs; read-only review access is checked the same way. These checks do not claim isolation from other same-user processes.

A frozen candidate is immutable; checks and reviews use it, never a mutable path with the same name. Before another writing attempt, the workspace must match its expected start or be rematerialized. Rejected candidates are stored with their evidence. Composition rules are pure functions in `realization` that compare exact bases, owned changes and registry changes and return a fixed plan. With one owner per file and one writer at a time, composition is a disjoint union; an overlap is returned as a Host error. `Materialize` applies the plan and reports what it observed; it never merges competing semantics or picks an owner.

## Check execution port

```go
// Package run; also used by final Apply verification.
type Checks interface {
	Run(context.Context, CheckRequest) (CheckReceipt, error)
}
```

`CheckRequest` binds a Host check ID, the exact review subject and an allowed check from fixed configuration, with its executable, arguments, environment, limits and tool identity. A reviewer may request an allowed check; a missing artifact test becomes a finding for its owner.

`CheckReceipt` carries the actual exit or termination state, output references, executable identity, candidate binding and observed drift or limits. A nonzero exit is a check failure; a startup or observation failure is reported separately. Cancellation never turns an unknown process into a pass. Checks run on a disposable materialization of the fixed candidate. An Agent saying "tests pass" is not a check receipt.

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

Publish writes into the working tree and never commits, merges or pushes; the owner's commit accepts the result. It revalidates the source guard at the write boundary, including additions, deletions and input changes, writes only the verified artifact and registry change and reads the result back. Its receipt distinguishes applied, verified no change, stale, partial, unknown and failed, with the completed paths, observed tree and registry identities and observation limits, also when an error is returned; callers keep such receipts. Multi-file publication is not atomic. Partial writes stay visible and recorded, and a later run captures the actual state. Even Applied is machine evidence, not human acceptance.

## Host lifecycle and completion

The coordinator runs one writer at a time with recorded ready and waiting tasks. A task moves through local work, checks and review; a rejection starts the next bounded submission. A Manager with children also composes their deliveries and obtains integration review before the Host accepts its subtree. The [execution contract](manager-apply-execution.md#one-run-state-machine) owns run states, outcomes and admission; this port defines no other lifecycle. Typed results keep the current state, any terminal outcome, reasons and observation limits even when a Go error is returned.

The root is ordinary in this protocol. `apply` adds the final evidence check and publication after the root delivery. Evidence stays bound to its subject; changed content or assumptions require current assessment. Source drift prevents publication even when every Agent reported success.

## Focused acceptance cases

The [walkthrough's refusal cases](manager-apply-library-flow.md#concrete-refusal-cases) are the first controlled-adapter checks. Add durable recording before dispatch, unknown dispatch and publication recovery, registry and tree mismatch, protected-input drift, exact duplicates and immutable review subjects. Then prove session start, Host calls, yielding, resumed input, fresh review and termination with the selected real executor, separately from the two real Applies.

Interfaces belong to their consumers, and adapters return concrete types ([Go interface guidance](https://go.dev/wiki/CodeReviewComments#interfaces)). Effectful calls take `context.Context` first ([Go context](https://go.dev/blog/context)). Pure model, organization and realization functions need no port just to be testable.
