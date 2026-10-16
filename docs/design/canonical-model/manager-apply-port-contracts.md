# Port contracts for the first reliable Apply

Date: 16 October 2026. Proposed P01 contracts derived from the [library delegation walkthrough](manager-apply-library-flow.md). The [Apply concept](manager-apply-concept.md) owns product behavior; the [Go architecture](manager-apply-go-architecture.md) owns package boundaries. These are proposed Go interfaces and behavioral contracts, not implemented APIs, provider wire formats or evidence of a successful Apply.

The [adapter selection and rebuild plan](manager-apply-rebuild-plan.md) now selects direct stdio App Server sessions and maps the old source to the replacement. These contracts remain provider-neutral.

The [configuration and mandate contract](manager-apply-configuration.md) and [journal/continuation contract](manager-apply-journal.md) extend this proposal with continuing assignments, source-bound mandates, versioned questions/answers and one SQLite-backed assignment stream. They also make the repository controller lifetime explicit through `apply.Control`.

## The exchanges before the interfaces

The root asks lending: “Reconcile lending and its reminders against M1, including code, tests and documentation. Coordinate the membership and copy-availability contracts.” It does not prescribe descendant file edits. Host validates the route and supplies complete responsibility, fixed inputs and the candidate base. Delegation returns a task ID immediately; the root yields and resumes later with a result.

Lending proposes its work. Host captures the actual file and registry changes as C1. A fresh reviewer might find that the implementation permits six loans although M1 permits five. Host sends that grounded finding to lending, captures the revised C2 and obtains a fresh review of C2. A reviewer pass on C1 cannot pass C2.

After the reminders child has delivered, Host composes the lending subtree and requests an integration review. Only Host can issue an accepted delivery to the root. The root still needs the other Package deliveries, system integration and final verification before Apply can publish.

## Values, identities and ownership

| Value | Owner and required content |
| --- | --- |
| Fixed tree and content reference | `snapshot`: explicit paths, bytes/content identities and relevant file modes. No repository acquisition or mutable workspace identity. |
| Definition reference | `model`: complete resolved v1 identity, with source provenance available for explanations. Runtime tasks do not introduce a second model identity scheme. |
| Manager identity | `organization`: one ordinary identity per local Package, including the root. Imported vocabulary does not create Managers. |
| Input binding | `run`: the fixed model closure and lock, Package inventory, configuration/custom instructions, ignore rules, initial registry, repository census and artifact source, plus selected executor/check profiles. `apply` constructs it from captured inputs. A model digest alone is insufficient. |
| Candidate reference | `realization`: immutable artifact tree and matching registry revision, with base provenance. It is not a workspace path. |
| Task and attempt | `run`: stable task ID, parent, Manager, operation, complete responsibility, outcome, model basis, input binding, base candidate and finite budget. Rework increments the attempt; it does not invent a new delegated responsibility. |
| Assignment and successor | `run`: a stable owner goal/grant, shared consumed allowance and linked fixed-basis runs. Successors have new task/session identities with explicit provenance; retained work does not inherit approval. |
| Effective mandate | `organization` renders source-bound facts; `run` binds the view to the current task/input. Reviewer assesses that authority basis under its own read-only grant. |
| Actor, session and step | `run`: actor role/scope, logical session ID and unique dispatch step ID. Host assigns IDs before effects. Provider handles remain adapter data, associated with those IDs. A Manager can have several sessions. |
| Review subject | `run`: input binding, exact candidate including registry, task/scope and purpose: local, integration or final verification. |
| Accepted delivery | `run`: Host-issued ID, exact subject, complete delivered scope, Definition assessments, child delivery IDs, review/check evidence and report. Agents cannot create accepted deliveries. |

Operational journal and scratch paths are excluded from source staleness only by an explicit fixed storage policy. Adding or deleting an ordinary file must affect the captured census. Do not silently exclude the whole repository or rely only on a Git commit while ignoring relevant working bytes.

The following names describe the central records. The [journal contract](manager-apply-journal.md) proposes versioned record codecs and assignment-scoped persistence. Identifiers are typed values, not arbitrary maps.

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
	Budget         Budget
}

type WorkProposal struct {
	Summary     string
	Assessments []realization.DefinitionAssessment
	Registry    []RegistryProposalID
	Children    []DeliveryID
	Questions   []QuestionRef
}

type ReviewProposal struct {
	Subject  ReviewSubject
	Verdict  ReviewVerdict
	Findings []Finding
	Evidence []EvidenceRef
}
```

Delegation's actual sender is taken from the bound session. It cannot choose its own authority, base, attempt or source binding. Host allows only a direct child and preserves the operation's full Package responsibility, inherited restrictions and unresolved obligations. A cited subset of Definitions does not narrow reconciliation.

Work proposals are explanations, not trusted diffs. Definition assessments cover every local Definition with a disposition, grounds and observable artifact/evidence references. Vocabulary-only requires explaining where concrete uses satisfy any obligations; it cannot conceal the policy limit. Registry proposals are Host-mediated and refer to authorized path claims. A no-change proposal needs the same assessment as a changed candidate.

A finding carries its affected scope, model grounds, candidate evidence and requested outcome. Review verdicts distinguish pass, changes requested, needs decision and failed assessment. Host validates structure, subject and evidence references; it cannot deterministically prove the reviewer's semantic judgment. A required unresolved model question blocks completion and cannot be repaired by artifact-only Apply.

`QuestionRef` binds identity and version. The application records actual owner answers against that exact proposal and basis; the Agent cannot self-assert owner authority. A context-only answer may continue the same run. Changed intent requires an authorized model operation and a new bound run, with shared allowance and selected retained work. See [questions and model changes](manager-apply-journal.md#questions-answers-and-model-changes).

## Application boundary and source acquisition

CLI and MCP call a concrete `apply.Service.Apply(ctx, request)` and receive a result plus an ordinary error. No additional service interface is needed for those callers. Apply depends on this consumer-owned acquisition port:

```go
// Package apply.
type Source interface {
	Capture(context.Context, CaptureRequest) (CapturedInputs, error)
}
```

`CaptureRequest` selects the repository, canonical root, explicitly supplied dependencies and operation configuration. `CapturedInputs` supplies fixed bytes/trees, the repository census, configuration, registry and ignore inputs, and an opaque publication guard. It includes the identities needed to reproduce those inputs; it does not guess missing dependencies from prose or fetch unspecified vocabulary.

Apply compiles those bytes with `model`, derives organization/context and validates initial file accounting. Unknown files need an explicit classification before they may be changed. Model, configuration and independent scenario expectations stay outside Agent write authority. Invalid required inputs stop work before dispatch. Store the bound inputs durably before launching a session.

`apply.Control` acquires the repository controller/store binding and supplies the run journal; its exact interface and lease lifetime are specified by the [journal contract](manager-apply-journal.md#physical-store-and-commit-contract). Apply revalidates captured inputs under that lease. CLI and MCP do not acquire execution authority or coordinate successor publication themselves.

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
| `SessionSpec` | Host session ID, actor role and Manager/task/attempt, executor profile identity, workspace access and explicitly supplied history. Review sessions start fresh; implementer conversation is not supplied as review context. |
| `StepRequest` | Unique step ID, session ID, task/subject binding, exact prompt/context/custom-instruction payload references, additional messages and effective tool grants. Host records it before dispatch. |
| `StepReceipt` | Step/session identity, observed status, proposed result or yield, output references, actual usage when observable and uncertainty/observation limits. Yield, proposed completion, needs decision, failure and uncertain interruption are distinct. None means accepted delivery. |
| `SessionQuery` | Known Host session ID and, when relevant, step ID. Inspection reports known dispatch/termination state and available observations; unavailable information stays unknown. |
| `SessionEvent` | Session/step, sequence/correlation, event kind and exact observed payload. Includes prepared outbound payloads, transport observations, incoming content and tool exchanges. |
| `ToolCall` / `ToolReply` | Correlation ID and a closed typed request/result for delegation, yielding, targeted reads, path claims, registry proposals, questions, allowed checks or result submission. Actual sender comes from the session channel. |

`SessionIO` is implemented by the Host. The adapter invokes `Observe` with the actual outgoing content before sending it and must receive durable acknowledgement before dispatch. The run stores both compiled instructions and actual observable transport input/output. Provider-supplied hidden context and unobservable activity are reported as limits; they are not invented from a template. A selected executor must support the required Host protocol and input observation.

`Call` validates role and current task state. A delegate call records a child task and returns its ID. The parent then yields; a child is scheduled outside that callback. A callback must not recursively run a session or wait for an entire child workflow. Reviewer tools expose read/check/report capabilities; they cannot claim paths, change registry ownership or delegate implementation.

The Host deduplicates a repeated correlation ID with identical content and returns its recorded reply; reusing it with different content is an error. Duplicate transport observations do not create new tasks or attempts. This is not a guarantee of exactly-once external execution.

Cancellation requests propagate through `context.Context`. A timeout or error after possible dispatch returns the known receipt and an uncertain state. Inspect the known session/step before retrying; do not send another implementation merely because a response was lost. `Stop` reports observed termination, not only that cancellation was requested. An uncertain writer blocks workspace handoff. Callback errors prevent further dispatch; any already possible external activity remains recorded as uncertain until reconciled.

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
| `WorkspaceSpec` | Host workspace ID, task/attempt, exact base candidate, protected input binding and role access. Host grants writes to existing owned paths and explicitly reserved new paths; parents do not supply a descendant edit list. |
| `WorkspaceObservation` | Known ID, materialized base, access, actual lifecycle state and recovery information. A partial/uncertain preparation still returns its known identity for inspection. |
| `FreezeRequest` | Workspace ID, expected base and current Host authority/registry ledger, with the completed writing step. Unknown active writers prevent freezing a deliverable. |
| `Harvest` | Full observed tree/delta, protected-input comparison, ownership violations and immutable candidate including registry when valid. The proposal's claimed file list cannot limit observation. |
| `MaterializeRequest` | New Host workspace ID and a fixed `realization.CompositionPlan`, expressed through snapshot/realization values rather than `run` task types. |

The Host owns claims and decisions. The adapter observes physical files. New paths require reservations before writing; retroactive registration does not authorize an unauthorized write. Freeze observes all relevant paths, including additions/deletions and protected model/configuration inputs. Read-only review access is also checked against actual changes. These checks do not claim OS isolation from same-user processes.

Freeze retains immutable candidate content independent of later workspace changes. Checks and review use that content, not a mutable path bearing the same name. Before a further writing attempt, the workspace must still match its expected starting state or be rematerialized explicitly. Store rejected candidates and their evidence as well as accepted ones.

Composition rules are concrete pure functions in `realization`. They compare exact bases, owned changes and matching registry changes, returning either a fixed plan or conflicts. `Materialize` applies that plan and reports observed output. It cannot silently merge competing semantics or choose a winning owner. Conflicts go to the accountable Manager through ordinary work/review. Parent integration changes do not grant child-file ownership.

## Check execution port

```go
// Package run; also consumed by final Apply verification.
type Checks interface {
	Run(context.Context, CheckRequest) (CheckReceipt, error)
}
```

`CheckRequest` binds a Host check ID, exact review subject/candidate and an allowed check specification from fixed project configuration. That specification supplies executable/arguments, relevant environment, limits and tool identity. A reviewer may request an allowed check; needing a new artifact test becomes a finding for its owner.

`CheckReceipt` carries the actual exit/termination state, stdout/stderr references, executable identity, candidate binding and observed input drift or limits. Nonzero exit is an observed check failure; startup/observation failure is separately reported through receipt and error. Cancellation does not turn an unknown process into a passing check. Execute on a disposable materialization of the fixed candidate and account for generated check outputs. No Agent's sentence “tests pass” becomes a check receipt.

## Durable journal port

```go
// Package run; apply uses the same run journal for its lifecycle.
type Journal interface {
	Put(context.Context, Payload) (BlobRef, error)
	ReadBlob(context.Context, BlobRef) (Payload, error)
	Append(context.Context, AppendBatch) (AppendReceipt, error)
	Read(context.Context, EventQuery) (EventPage, error)
}
```

Payloads are immutable content-addressed bytes whose typed references retain media/encoding information; reads verify their identity. An append batch supplies assignment ID, unique batch ID, expected stream revision and typed events. Each event binds applicable run, task, actor, session, step, attempt, candidate and causal predecessor. Referenced payloads must be durable before append acknowledgement. Assignment scope permits questions, inherited budget and successor creation between runs without rewriting earlier inputs.

Append is one logical commit under a single coordinated Host writer. A repeated batch ID with identical bytes returns the original receipt; different bytes are rejected. A stale expected revision is a conflict and stops dispatch. An uncertain append is reconciled by reading the known batch before an external effect. Read returns an ordered committed prefix with a stable revision and pagination. The [SQLite journal adapter](manager-apply-journal.md) now specifies physical storage and the explicit controller lease; no event bus is required.

Record task creation, exact input construction, dispatch, observed tools and extra reads, questions/replies, candidates/deltas, assessments, review findings, rework, composition, checks and publication. Reports are compact projections referencing those records. Required journaling failure prevents further work; it does not erase a previously possible external effect. Append/dispatch are not a distributed transaction.

## Guarded publication port

```go
// Package apply.
type Publisher interface {
	Preflight(context.Context, PublicationRequest) (PublicationPermit, error)
	Publish(context.Context, PublicationPermit) (PublicationReceipt, error)
	Inspect(context.Context, PublicationID) (PublicationReceipt, error)
}
```

`PublicationRequest` contains a Host publication ID, original acquisition guard/input binding, the exact verified candidate with matching registry, final evidence references and the artifact-only write target. Preflight checks readiness and source state; its permit is bound to that request and does not grant indefinite permission.

It also carries the live repository control check, a run-scoped Host authorization check and the run/epoch binding. A live controller alone does not authorize an obsolete run. Serialized history records the binding value, not a restorable live capability. Supersession invalidates a predecessor's unused permit. Retained private work remains distinct from the actual captured repository source used by the publication guard.

Publish revalidates the original source guard at the actual write boundary, including relevant additions/deletions and input changes. It writes only the verified artifact/registry change and reads back the result. Its receipt distinguishes applied, verified no change, stale, partial, unknown and failed. It includes actual completed paths, observed tree/registry identities and read-back/observation limits, including when an error is also returned. Callers must preserve such receipts rather than discard them on non-nil error.

Multi-file publication is not promised atomic. Partial writes remain visible and recorded; Apply does not announce success or rerun blindly. Inspect the known publication ID to establish recovery state. A subsequent operation must capture the actual state. No-change publication also requires current source/evidence and read-back. Even a complete Applied result is machine evidence, not human acceptance.

## Host lifecycle and completion

The initial coordinator schedules one writer at a time, with recorded ready/waiting tasks. A task progresses through prepared, working, proposed, checking, reviewing and reviewed-local states; changes requested return to working with the next bounded attempt. A Manager with children additionally waits, composes and obtains integration review before Host accepts its subtree. A leaf still needs its complete local assessment. Needs-decision, exhausted budget, failed and uncertain states cannot be treated as completed work.

The root is ordinary in this protocol. `apply` adds final aggregate verification and publication after the root delivery. Local review, subtree integration review and final verification are separate purposes. Evidence remains bound to its original subject; changed content or assumptions require current assessment. Full reconciliation is the initial policy, without speculative semantic reuse. Source staleness prevents publication even if every Agent reported success.

## Focused acceptance cases and remaining P01 work

The [walkthrough's refusal cases](manager-apply-library-flow.md#concrete-refusal-cases) are the first controlled-adapter contract checks. Add journal durability-before-dispatch, unknown dispatch/publication recovery, registry/tree mismatch, protected-input drift, exact duplicate handling and immutable review subject cases. Then prove session startup, Host calls, yielding, resumed input, fresh review and termination with the selected real executor. Neither those tests nor this design substitutes for M1-to-A1 and M2-to-A2 with real Agents.

The [rebuild plan](manager-apply-rebuild-plan.md) supplies the first executor choice and source removal map; the [configuration](manager-apply-configuration.md) and [journal](manager-apply-journal.md) supply the proposed settings, records and lifecycle. P02's evaluation interface/protected expectations and the exact executable/schema capability gate remain before real Agent work. Implementations still need complete codec validation and focused conformance evidence; these proposals do not mark runtime readiness.

Interfaces are owned by their consumers; adapters return concrete implementations. This follows [Go's interface guidance](https://go.dev/wiki/CodeReviewComments#interfaces). Effectful calls take `context.Context` first and propagate cancellation while reporting what actually happened ([Go context](https://go.dev/blog/context)). Pure model, organization and realization functions need no port solely for testing.

Return to the [library walkthrough](manager-apply-library-flow.md), [Go architecture](manager-apply-go-architecture.md) or [implementation plan](manager-apply-implementation-plan.md).
