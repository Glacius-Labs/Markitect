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
| Accepted delivery | `run`: Orchestrator-issued ID, exact subject, delivered scope, Definition assessments, child delivery IDs, review and check evidence, report. Agents cannot create one. |
| Task dependency | `run`: the producer task and the accepted delivery the dependent waits for. The dependent's step sees it in the current aggregate and records that exact version ([D3](manager-apply-core.md#deliveries-and-composition)). |

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

The sender of a message is the bound Manager session; it cannot choose its own authority, base, attempt or source binding. The Orchestrator routes the message along tree edges ([M1–M3](manager-apply-core.md#messages)) and preserves each recipient's complete responsibility, resolved constraints and unresolved obligations. A message cannot create participation, replace the operation or narrow reconciliation to a cited subset of Definitions.

Work proposals are explanations, not trusted diffs. Definition assessments cover every local Definition with a disposition, grounds and evidence references; a `vocabulary` disposition must show where concrete uses meet their obligations. Registry proposals refer to Orchestrator-authorized path claims. A no-change proposal needs the same assessment as a changed candidate.

A `Finding` has an Orchestrator-recorded ID, a severity (blocking or advisory), the affected responsibility, its model and input basis, the requested evidence and the originating reviewer; a candidate reference is optional for concerns raised before a candidate exists. A `FindingStateChange` moves an existing finding to resolved (with current evidence) or not applicable (with grounds), as defined in the [execution contract](manager-apply-execution.md#evidence-need-survives-budget-pressure). Only a bound independent reviewer can issue one. Verdicts distinguish pass, changes requested, needs decision and failed assessment. The Orchestrator derives the open findings from task, child and predecessor history and validates provenance and bindings; whether evidence suffices remains the reviewer's judgment.

A dependent task implements against a producer's result only once the Orchestrator has accepted it; its next step sees the result in the current aggregate ([D3](manager-apply-core.md#deliveries-and-composition)). Step and workspace records bind what was actually supplied: prompts, context, the aggregate version, continued-session history and later replies.

`QuestionRef` binds identity and version. The application binds owner answers to that exact proposal; an Agent cannot assert owner authority. A context-only answer may continue the same run; changed intent ends the run Incomplete and needs a committed model edit and a continuation ([journal](manager-apply-journal.md#questions-answers-and-model-changes)).

The illustrative `TaskSpec.Outcome` above is the static Orchestrator operation instruction, not parent-authored prose. Its model basis and complete scope are compiler-derived. Additional work messages remain separate and cannot redefine that mandate. Exact responsibility and prerequisite records must follow the remaining [process details](manager-apply-concept.md#remaining-process-detail-and-technical-boundary); existing provider Session contracts are not a Manager scheduler.

The confirmed process uses normal deliveries with limited scope while their Manager remains open. The technical representation binds Orchestrator acceptance ([D1](manager-apply-core.md#deliveries-and-composition)) separately from a reviewer pass. Do not invent a parallel receipt lifecycle or a budget reset for each limited delivery. Successful deliveries consume shared execution capacity but neither use nor reset the rejection counter, whose identity is fixed per run, Manager and review purpose by the [execution contract](manager-apply-execution.md#admission-amendments-and-finite-correction). Exact counting, binding and recovery need implementation evidence.

## Bounded child reports

The owner selected this target message structure; it is not an implemented wire schema or a second delivery ledger. Reports reach the parent through the Orchestrator, starting with the Explore report of [T3](manager-apply-core.md#manager-tasks), and the journal records them. The Orchestrator binds sender, recipient, run, Manager and exact step; a child cannot assert those identities or issue an accepted delivery.

| Part | Required meaning |
| --- | --- |
| Identity and basis | Report ID/version, exact originating step and workspace/candidate binding, and any accepted delivery reference. A later report identifies what it supersedes. |
| Responsibility and status | Covered scope and initial assessment, progress, limited delivery or completion proposal. Remaining responsibility stays explicit; only the Orchestrator can confirm completion. |
| Result and interfaces | Short result, changed interface references and implications for the direct parent. No descendant transcript dump. |
| Prerequisites and replies | Needed results, exact producer/delivery references when known, outstanding questions and replies required to proceed. |
| Findings and uncertainty | Open finding references, unresolved assumptions and observation limits. Absence from a summary never resolves an existing finding. |
| Evidence and detail | Exact immutable evidence/detail references with the scope each supports. An Agent's report remains a proposal until required review and Orchestrator acceptance. |

The Orchestrator enforces finite report and assembled-input byte limits from a versioned policy. Exact field names and numeric defaults are producer work, not additions to today's execution YAML. Keep the complete direct-child census and outstanding obligations in the existing journal; only bounded views enter a turn. If a report or parent input exceeds its limit, split it visibly into version-bound parts with stable ordering, an inspectable index and an explicit remainder. Existing message and step observations record which parts were supplied; a changed report version requires current coverage. There is no separate paging ledger, and a delivery acknowledgement does not prove semantic understanding. Required findings and prerequisites cannot be silently truncated or disappear behind a successful summary. If required material cannot be supplied within the finite limits, retain incomplete work and report that boundary.

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
| `SessionSpec` | Orchestrator execution-session ID, stable Manager identity, known conversation identity or explicit fresh-conversation handoff, role, executor profile and runtime image, fixed workspace access/authority ceiling and supplied history references. Resume capability is established, not inferred. Review sessions start fresh. |
| `StepRequest` | Unique step ID, execution session and conversation association, task and exact admitted workspace version, the aggregate version it starts from and own starting changes ([D3](manager-apply-core.md#deliveries-and-composition)), subject binding, exact prompt/context/instruction references, additional messages and permitted effects. Recorded before dispatch. |
| `StepReceipt` | Step and session identity, observed status, the proposed step outcome ([T3](manager-apply-core.md#manager-tasks), [T5](manager-apply-core.md#manager-tasks), [T6](manager-apply-core.md#manager-tasks)), output references, usage when observable and observation limits. A report, a submission, a `wait`, a step without a valid outcome and an uncertain interruption are distinct; none is an accepted delivery. |
| `SessionQuery` | Known session ID and, where relevant, step ID. Unavailable information stays unknown. |
| `SessionEvent` | Session and step, sequence and correlation, kind and exact observed payload, including prepared outbound payloads, transport observations, incoming content and tool exchanges. |
| `ToolCall` / `ToolOutcome` | Correlation ID and a closed typed request for one tool of the role's [Orchestrator tool catalogue](manager-apply-core.md#host-tools). `Call` records it, the Orchestrator answers it and the answer is recorded before the adapter returns it unchanged: `ok` with a result, or `refused` with a reason, which the adapter reports as a tool error. The write that delivers an answer names its call and may happen once. The sender is the session whose container channel carried the call. |

A step's output contract (`application/vnd.markitect.step-contract+json`, format `markitect.step-outcome/v1`) names its phase and the Orchestrator tools that end it; the outcome reaches the Orchestrator only through those tools, and the adapter does not interpret the turn's final text. The two session adapters still read the earlier Explore contract; moving them to this contract is the Coordinator's follow-up. The session plan's tool catalogue (`application/vnd.markitect.tools+json`) lists exactly the tools of the step's phase ([Orchestrator tools](manager-apply-core.md#host-tools)), and the adapter registers exactly those. Every session runs in its own container through the session helper `markitect-agentd`, which serves the Orchestrator tools to the provider CLI over MCP and carries everything over one stdio connection ([rebuild plan](manager-apply-rebuild-plan.md#container-runtime)). The Orchestrator implements `SessionIO`. The adapter calls `Observe` with the actual outgoing content and waits for durable acknowledgement before sending it. Hidden provider context and unobservable activity are reported as limits, never reconstructed from templates. An executor that cannot support this protocol and its input observation is not selectable.

The records above describe the selected target; the implemented Go API is `run.Sessions` and `run.SessionIO` in source. In the first strategy one execution session is one container serving exactly one step, so the session plan's workspace and candidate are the step's. The plan also names a provider-home volume that survives the container and, when continuing, the provider conversation to resume; a resumed session binds only under that identity, and reviewers never resume. The admissions hand the adapter the exact recorded profile, tool, input and contract bytes. A source-only continuation claim is not a physical capability.

Nothing changes a workspace while its step runs, and the next step starts from a newly materialized version ([S6](manager-apply-core.md#steps), [D3](manager-apply-core.md#deliveries-and-composition)). Every action is checked against the active step and version. A late request from an older step cannot write or create new current evidence; a repeated completed request returns only its old receipt. Independent review uses its frozen subject rather than a Manager's workspace.

`Call` validates role, active step/version, permitted effects, task state and the message route ([M1](manager-apply-core.md#messages)). Before the complete Explore census, all artifact mounts are read-only and the Orchestrator denies artifact-mutating tools and workspace effects; role alone cannot grant them. Read-only analysis and proposed classifications remain distinct from mutation. Initial operation propagation is an Orchestrator obligation independent of tool calls or Agent output. A Manager can send messages and ends its step with one outcome ([T5](manager-apply-core.md#manager-tasks)); a callback never runs a session recursively or waits for a child workflow. Reviewers can read, request allowed checks and report; they cannot claim paths, change ownership or control Manager participation.

The core protocol owns the closed [Orchestrator tool catalogue](manager-apply-core.md#host-tools); each role sees only its own. DEC-039 removed the former `plan` and `delegate` controls. Owner answers arrive only through the application.

Tool deduplication binds the actual execution session, channel, step and call identity. An exact repeated request with identical content returns its original recorded reply; changed content under that identity is an error. A reused provider call ID in another step or channel is not evidence of the same invocation and must not inherit its receipt or authorize replay of unresolved earlier work. An uncertain original call stays uncertain until reconciled. Duplicate observations create no tasks or attempts. Cancellation propagates through `context.Context`. A timeout or error after possible dispatch returns the known receipt with an uncertain state; inspect the known session and step before retrying, and never resend work just because a reply was lost. `Stop` reports observed termination, not only a cancellation request. An uncertain writer blocks workspace handoff. A `SessionIO` callback error prevents further dispatch; external activity that may already have started stays uncertain until reconciled.

## Candidate workspace port

The workspace adapter only materializes and observes. `run` decides what a workspace holds ([D3](manager-apply-core.md#deliveries-and-composition)) and what its observed change means ([S4](manager-apply-core.md#steps)).

```go
// Package run.
type Workspaces interface {
	Materialize(context.Context, WorkspaceSpec) (WorkspaceObservation, error)
	Harvest(context.Context, HarvestRequest) (WorkspaceHarvest, error)
	Close(context.Context, WorkspaceID) (WorkspaceObservation, error)
}
```

| Record | Required fields and meaning |
| --- | --- |
| `WorkspaceSpec` | Orchestrator workspace ID and the exact tree: the current aggregate plus the Manager's own unaccepted changes. |
| `WorkspaceObservation` | ID, the runtime volume that a session plan mounts, the identity of the content read back after materialization, whether the volume is closed, and any uncertainty with its reason. |
| `HarvestRequest` | Workspace ID and the entry and byte limits of one observation. |
| `WorkspaceHarvest` | The complete observed tree after the step's container stopped. Entries that a tree cannot represent are listed by name instead of being dropped. An observation that hit a limit is uncertain. |

The Orchestrator owns claims and decisions, and the adapter only observes files. `run` compares a harvest with the step's start tree, checks owned and reserved paths and protected inputs, and records the result. A later claim never authorizes an earlier unauthorized write. `run` stores harvested candidates as payloads in the Orchestrator's private journal store.

Recovery rests on three rules:
- `Materialize` refuses an existing ID, so it never overwrites content that was not harvested.
- `Harvest` is uncertain while any container still mounts the volume.
- `Close` is idempotent and reports an absent volume as closed.

Observation follows Git's view of files. A regular file is executable exactly when its owner execute bit is set, other permission bits are ignored, and empty directories are ignored. Only non-regular entries and names that a snapshot path rejects are unsupported. A harvest excludes exactly what the repository census excludes: Git administration and `.markitect/state`.

Workspaces are volumes of the container runtime. The adapter materializes and harvests them inside a helper container, so observation follows Linux file semantics. A session container mounts only its own workspace.

Checks and reviews use frozen candidates, never a mutable workspace with the same name, and rejected candidates are stored with their evidence. Composition is a pure function in `realization` that checks the [D2](manager-apply-core.md#deliveries-and-composition) conditions and returns a fixed result. It never merges competing content or picks an owner.

## Check execution port

```go
// Package run; also used by final Apply verification.
type Checks interface {
	Run(context.Context, CheckRequest) (CheckReceipt, error)
}
```

`CheckRequest` binds:
- an Orchestrator check ID and the exact tree to check;
- the runtime image and the absolute executable inside it;
- literal arguments: the profile's prefix followed by the project arguments, never passed through a shell;
- unique `KEY=VALUE` environment entries from the profile's fixed policy;
- the working directory as a boundary inside the tree;
- container limits for CPU, memory and processes, a timeout and an output limit.

A reviewer may request an allowed check. A missing artifact test becomes a finding for its owner.

`CheckReceipt` carries:
- the image that ran and the identity of the tree copy it ran on;
- whether the process started and exited, and its exit code;
- whether it timed out;
- the combined output up to the limit, with a truncation mark;
- any uncertainty, with its reason.

A receipt passes only for exactly its request: same check, image and tree, with an observed zero exit that neither timed out nor is uncertain. A nonzero exit is a check failure, and a startup or observation failure is reported separately. Cancellation never turns an unknown process into a pass. Checks run in a container from the check profile's image, on a disposable copy of the fixed candidate. An Agent saying "tests pass" is not a check receipt.

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
	Preflight(context.Context, PublicationRequest, ControlCheck) (PublicationPermit, error)
	Publish(context.Context, PublicationPermit, ControlCheck) (PublicationReceipt, error)
	Inspect(context.Context, PublicationRequest) (PublicationReceipt, error)
}
```

`PublicationRequest` holds:
- a publication ID and the run;
- the repository, which is the same explicit acquisition location the capture used;
- the original acquisition guard;
- the exact artifact and registry changes against the captured tree.

An empty change set is valid and publishes nothing. The controller fence is passed separately as a `ControlCheck`, which a `ControlSession` satisfies.

`apply` derives the changes from the verified aggregate and limits them to the paths that the artifact-only target permits. It also voids the permit of a finished run, because the adapter knows no runs. Preflight rechecks guard and controller and refuses changes the target cannot represent, such as case aliases or modes the repository does not track. Its opaque permit is bound to that request.

Publish writes into the working tree and never commits, merges or pushes; the owner's commit accepts the result. It checks the guard and the controller before its first and after its last write, including additions, deletions and input changes. It writes exactly the requested changes, the way Git checks files out in that repository. It reads every written path back in Git's clean form, so line-ending conversion does not count as a mismatch.

Its receipt carries one outcome:

| Outcome | Meaning |
| --- | --- |
| applied | Every change was written and read back, and the check after the last write held. |
| no_change | The change set was empty and the guard held. |
| stale | The guard failed before any write. |
| partial | Some changes were written, not all. |
| unknown | The state of some path, or of the check after the last write, is not known. |
| failed | A definite failure, with nothing written. |

The receipt also classifies every requested path:
- it holds the requested content;
- it still holds the captured content;
- it holds something else or could not be read.

After the writes, the guard holds when every path outside the change set is unchanged. The receipt states what else is known, also when an error is returned, and callers keep such receipts. `Inspect` classifies the paths of an earlier request, which `apply` journaled before `Publish`, after an interruption. Multi-file publication is not atomic. Partial writes stay visible and recorded, and the owner commits or reverts them before a later run captures the state. Even an applied result is machine evidence, not human acceptance.

<a id="host-lifecycle-and-completion"></a>

## Orchestrator lifecycle and completion

The Orchestrator guarantees the same operation reaches every local Manager and runs one step at a time ([S5](manager-apply-core.md#steps)) with recorded ready and waiting tasks. The ready-work policy is fixed by the product, not supplied by an Agent. No silent skip or pass-through startup exception is allowed. A task moves through the states of the [task machine](manager-apply-core.md#manager-tasks); a rejection starts the next bounded submission. A Manager with children also submits an integration candidate on the composed aggregate of its subtree and obtains integration review before the Orchestrator accepts its subtree ([T9](manager-apply-core.md#manager-tasks), [T13](manager-apply-core.md#manager-tasks)). The [execution contract](manager-apply-execution.md#one-run-state-machine) owns run states, outcomes and admission; this port defines no other lifecycle. Typed results keep the current state, any terminal outcome, reasons and observation limits even when a Go error is returned.

Explore and the write barrier follow [T1](manager-apply-core.md#manager-tasks) and [T2](manager-apply-core.md#manager-tasks), and admission follows [T4](manager-apply-core.md#manager-tasks). Local planning thereafter has no global barrier. Separate parent hints preserve the operation, compiled context and authority. These are admission requirements within the existing lifecycle, not additional run states.

The root is ordinary in this protocol. `apply` adds the final evidence check and publication after the root delivery. Evidence stays bound to its subject; changed content or assumptions require current assessment. Source drift prevents publication even when every Agent reported success.

## Focused acceptance cases

The [walkthrough's refusal cases](manager-apply-library-flow.md#concrete-refusal-cases) are the first controlled-adapter checks. Add durable recording before dispatch, unknown dispatch and publication recovery, registry and tree mismatch, protected-input drift, exact duplicates and immutable review subjects. Then prove session start, Orchestrator calls, step outcomes, conversation resume in a new container, fresh review and termination with the selected real executor, separately from the two real Applies.

Interfaces belong to their consumers, and adapters return concrete types ([Go interface guidance](https://go.dev/wiki/CodeReviewComments#interfaces)). Effectful calls take `context.Context` first ([Go context](https://go.dev/blog/context)). Pure model, organization and realization functions need no port just to be testable.
