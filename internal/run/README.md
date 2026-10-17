# Run admission

This package owns pure decisions over a fixed Apply basis and each assignment's
shared ledger on one repository stream. It consumes model, snapshot, organization and
realization values. The caller supplies acquired original bytes, static profile
facts, clock labels and elapsed durations. None establishes runtime capability or
owner authority by itself.

`Bind` retains the complete supplied model closure and lock, compiled content and
Package inventory, actual census with modes, initial registry, classifications,
protections, source guard, configuration with selected ceilings and provenance,
ordered instructions and static profiles. It requires the existing realization
accounting to accept the complete initial census. Source-bound configuration
resolution and source acquisition remain their producers' duties. The caller
must supply every input document, including strategy, registry and ignore inputs;
run does not discover files or decode YAML.

`Propose` is pure. `Commit` writes original payloads, then appends an atomic batch
using its expected stream revision. Only the exact successful acknowledgement
adopts the proposed projection, including Prepared and its run charge. An error
or mismatched acknowledgement leaves the last confirmed projection visible and
blocks dependent transitions. `ReconcilePending` checks the exact known batch
and original payloads. It never retries an external effect.

The four-method `Journal` is consumer-owned. Batches have a closed versioned JSON
codec and immutable identities. Identity checks precede CAS for exact retries.
The test journal demonstrates only process-local contract behavior. A physical
adapter must establish durable atomic visibility, blob retention and controller
fencing separately.

`AppendBatch.BindingRefs` lists its direct basis references;
`BindingOwnerRef` validates each original basis and derives the required owner
input reference through the same recorded codec as restoration. A store verifies
and retains both sets of bytes before Append acknowledgement, using these pure
APIs and `Payload` identity checks. No adapter-owned binding decoder or reducer
is needed. The v1 stream range is zero through `MaxRevision` (2^63-1), and both
interval endpoints must be exact batch boundaries. Exact batch lookup reports
the current observed head, which may be later than the matched batch's receipt.

`Restore` opens existing or genuinely empty history through bounded `Read` and
`ReadBlob` calls. Its explicit limits bound pages, batches, events and bytes,
including required blobs. Exhaustion returns `ErrRestoreLimit`, separately from
invalid history. Every page keeps the first committed head. The same
transition rules regenerate each complete batch; gaps, partial batches, changed
heads, invalid records and missing or changed blobs return no Coordinator.
The historical versioned binding, compiled digests, inventory and original owner
payload are verified without reinterpreting old input with today's compiler or
resolving settings again. Store/controller identity remains the Host's duty.

New preparations record basis v3. V3 adds the immutable static operation,
source-bound scope selectors and context limit, and an initial Explore policy.
V1/v2 retain their original recorded meaning and canonical bytes. Each passed compiled model contributes its
complete own Package inventory and every Definition's identity and owning model
ID, including imported models and explicitly empty inventories. `BindingInventory`
returns a basis-bound recorded view with copied model IDs, digests, manifest
references, Packages and Definition identities. It supplies no `model.Result`,
Definition bodies or semantic compilation claim. Validation checks recorded
shape, owning-model and Package membership, unique ordered inventories, raw
source relationships and agreement with the local Manager inventory. Original
compiler completeness and semantics remain producer assertions.

Literal v1 basis and batch bytes still restore and retain their original
references and retry receipts. Their imported Definition inventories are absent;
`RecordedInventory.Models` returns `ErrInventoryUnavailable` rather than an
invented complete empty inventory. These versions add no blob references or
Journal methods. The existing 64 MiB payload ceiling and explicit Restore byte
budget include the inventory. Work readiness and execution remain unavailable; the initial Explore protocol below
records participation only.

Restoration never calls Put or Append and never replays an external effect. Open
reservations keep their complete holds. `RecoveryRequired` is a separate Host
barrier, while `Current` still reports the actual committed run state. Only
explicit reconciliation observations or a stop request can proceed through that
barrier; every hold needs a confirmed settlement or proven non-dispatch release.
An identical recorded command or batch returns its original receipt without
another append or charge. An absent old proposal at an advanced revision cannot
be treated as an automatic retry; new decisions and commits remain explicit.

The current first slice handles Prepared, Running, quiescent Waiting,
Reconciling, and immutable non-successful Finished outcomes. Reservations are
recorded invocation intent; this package dispatches nothing. Unknown effects keep
their full holds. Explicit Host observations establish settlement or proven
non-dispatch. Writing preserves the completion reserve; assessment may use it.
An actual overrun is retained and blocks further admission.

`ExtendAssignment` records absolute cap targets and exact raw UTF-8 owner input
(at most 64 KiB). At least one cap must increase, none may decrease, and every
target stays within the assignment's original project ceilings. Completion
reserve and timeouts stay fixed. Consumption, invocation holds and observed
overruns remain unchanged. Each invocation retains the cap revision admitting
it; extension records retain every prior cap revision and original owner input.
These Host observations do not authenticate authority.

`CloseAssignment` requires a Finished current run, no open invocation, exact raw
owner input and a separate bounded Host quiescence observation. An empty hold
ledger, cancellation acknowledgement or stop request is insufficient. RUN
records the supplied observation; no physical termination is established here.
Closure keeps the last run's outcome and cannot reopen the assignment.

`Assignments` returns copied closed ledgers followed by the current assignment,
with original owner references, initial/current policy, cap revisions, runs,
invocations, extension inputs and closure observations. A distinct ordinary
assignment can be Prepared only after prior closure, with no predecessor. It has
its own budget and owner input; previous consumption, grants, holds and remaining
capacity never carry over. Run and invocation identities cannot be reused.
Exact historical command retries return the original receipt, even after a new
assignment starts. Restore rebuilds all of this history through the same reducer.

An open assignment supports continuation after a changed basis or a recorded
extension since the last run was Prepared. Its original owner input and bound
policy stay fixed; recorded cap targets change the shared ledger only. Each Prepared run
needs an available run unit and the new basis's structural review floor in the
remaining shared ledger; this need not replenish the configured reserve that
assessment work may spend.
The consumer-owned `Sessions` port maps one Host session to Open, one compiled
turn to Step, exact known provider handles to read-only Inspect, and bounded
termination to Stop. It names no engine or provider protocol. In the first
strategy one Session is one container serving exactly one Step; the Host
prepares a new Session for every Step. The plan names the step's workspace
volume, a separate provider-home volume that keeps the conversation across
containers, and optionally the provider conversation to resume. A resumed
Session binds only under that identity. Reviewers are read-only and never
resume. Session and step preparation commit their finite admission reservations
atomically with the preallocated identities, fixed profile/image/workspace and
input/contract references. `SessionAdmission` and `StepAdmission` read and
verify the recorded profile, tool, input and contract bytes from the journal and
hand them to the adapter, which uses exactly those bytes. Credential fields are
references only; the adapter resolves them at Open.
Each Step explicitly asserts the still-current Session plan binding, including
profile, image, workspace and candidate. A changed or unprovable binding needs a
new session. Physical workspace currentness remains the Host's responsibility.

`Open` may return acknowledged control readiness before the provider supplies
its actual session ID. `SessionIO.Ready` records that incoming acknowledgment
with a separate receipt; the Session is live, ready and unbound. Empty IDs and
requested UUIDs never imply readiness or binding. Open observations distinguish
ready, actually bound and uncertain outcomes; `Bind` cannot consume readiness.
The adapter records each actual send outcome before forwarding an immediate
incoming readiness/init event, buffering that observation within its limits.

Acknowledged readiness permits startup's observed elapsed time to settle, but
never makes it undispatched or terminated. Both Manager and reviewer require
that settlement before admitting one first Step on the still-unbound Session.
Its identity and finite reservation commit together. Only that Step's first
non-control send can receive a fresh permit before actual binding; even a
`NotSent` observation does not restore this initial slot. Known preparations
return their receipts without another send. Control before binding is bounded
termination during reconciliation, not a way to relabel user input. Adapters
still own truthful protocol interpretation and encoding of the compiled input.

The actual provider init then binds the observed ID before definite Step/tool
observations and later Steps. Missing init, uncertain send or append failure
retains effects and holds; another Open cannot replace a recorded startup send.
Restoration preserves readiness and the first Step/send identities behind the
existing recovery barrier, without dispatching anything. Readiness cannot erase
earlier uncertainty or supply inspection handles.

`SessionChannel` implements the serialized Host `SessionIO` callbacks against
the same Coordinator and journal. Before every provider write, including tool
responses and notifications, `PrepareSend` must return a fresh acknowledged
`SendPermit` whose identity matches the actual bytes. Failed or uncertain
acknowledgement never grants dispatch. Re-observing a preparation never grants
another send, even after restart. Actual sent/not-sent/uncertain observations
remain separate. Shutdown control sends stay available during reconciliation
without clearing earlier unknown activity or charging new Agent work.

Wire records separate the actual byte digest/length from the permitted journal
representation. The Host redacts known secrets **before** supplying any payload
for append; RUN receives no secret value and cannot perform that redaction.
Redacted records retain their explicit observation limit and cannot reproduce or
authorize replay of the original wire. `AppendBatch.SessionRefs` lists exactly
the direct stored payloads, alongside `BindingRefs` and `BindingOwnerRef`; the
store verifies and retains all three closures before acknowledging Append.

`Call` records a tool call, lets the channel's `ToolHandler` answer it and
records the answer as a closed `ToolOutcome` (`ok` with a result or `refused`
with a reason) whose encoding fits the session's wire limit. The adapter
returns those exact bytes to the provider; a refusal is an answer, not a
transport failure. While the run admits no new work (recovery, a stop request
or any state but Running), the call is answered with a fixed refusal and the
handler is not called. Calls are deduplicated by session channel and provider
call identity, including the exact step, tool identity and
content/representation: an exact repeat returns the recorded answer without
executing again, and a changed call conflicts. A handler error or an invalid or
oversize outcome leaves the call pending; it cannot execute again implicitly.
If recording the answer fails, `Call` returns the error together with the
outcome, which the adapter must not send; once reconciliation shows the answer
absent, the Host records it through `ToolReconciliation`. Recorded answers are
never redacted and always carry the tool-outcome media type. Slices without
Host tools use `RefuseTools`; the Manager and reviewer tools arrive with the
core protocol.

Before the Claude Code adapter relies on resume binding, a bounded probe must
show that a resumed Claude Code session reports the resumed session ID in its
init event. The runtime capability record shows that the conversation content
continues, not the reported identity.

Provider session/step IDs remain distinct from Host IDs. Inspect can select only
the exact journal-bound handles and never creates or resumes a provider Session
or turn. A future separately authorized bounded read-only history helper may
use a container; it is not implemented here and changes no Inspect contract.
An unavailable history reader remains an explicit observation limit. Stop
records actual termination or uncertainty, preserving partial observations on
error. Settled startup time does not mean a provider stopped: live sessions,
active steps and unresolved Host tool activity prevent run completion and
assignment closure. A stop request or callback cancellation does not clear them.
Restoration retains the recovery barrier until explicit reconciliation and
elapsed settlement; no port is executed during Restore. Proposed terminal output
is retained evidence, not an accepted delivery.

Review and task execution, final evidence, Verifying/Publishing transitions and
Applied/NoChange are not implemented yet. There is no concrete Session,
Workspace, Check, SQLite, provider or Apply execution adapter here. The full
contracts remain in the design documents.

## Initial Explore policy

Every new live basis uses v3. The complete recorded local topology queues root
and every direct parent-child edge, including empty and intermediate Packages.
`PrepareExploreSessionCommand` associates one queued Manager with an initial
read-only session. `PrepareExploreStepCommand` recompiles supplied original model
bytes, resolves settings, checks the complete binding against the current run,
and calls the actual organization selection and renderer. Scope and context
limits come only from that bound input. The root receives the original owner
request; registered own artifacts come from the fixed census, with binary
content identified without claiming it was read as text. No parent-authored
operation, participant list or inferred relationship selects the roster.

There is one initial step per Manager. Sessions are sequential, physically and
semantically read-only under the Host's still-unproved isolation obligation.
Their expense class remains `Writing`, which protects completion capacity; it
is not a writer grant. This existing cost/access coupling is a bounded slice
limitation, not the intended final session-continuity contract. There is no
basis advance, resume capability or working-candidate transition here.

`InitialExplore` distinguishes queued, prepared, sent-observed, attributed step,
report and quiescence observations. A transport write is not provider acceptance.
The closed inline input/report identities bind the exact Manager, parent, run,
operation, context, session, step and input. Reports can include uncertainty and
questions. They establish neither prerequisite satisfaction nor a reviewed
no-change result, accepted delivery or completed Manager. Missing or malformed
terminal output is retained with its observation limit and cannot count; this
single-step slice supplies no automatic replacement attempt.

The initial gate clears only after every Manager's report is confirmed and all
effects are stopped/reconciled and charged. A terminal step alone is insufficient.
An unexpected tool call is recorded and refused; no Host tool runs in this
slice. Unresolved callbacks and transports remain
visible even after a session stops. All unclassified sessions, later steps,
standalone invocation reservations and unimplemented work/review/tool phases
remain denied after the gate clears. Actual Workspace/Check/Apply execution and
provider-output extraction are absent; source fixtures prove no real dispatch.

V3 protects two session starts and two steps per required review subject: one
local review for every Manager, plus integration review for every Manager with
children. Distinct physical phase bindings are conservatively budgeted; no
unproved session reuse discounts this minimum. The original v1/v2 basis floors
still replay; session histories recorded before the step-bound session port are
not supported. Exact
known retries return their original receipts before fresh-admission checks.
Restored legacy history permits only known-effect observation/reconciliation and
stop, not fresh legacy execution or replaying a send.

Operation/context/contract/report bytes are inline in the basis or existing
session payloads. `SessionRefs` retains their complete direct blob closure; no
additional evidence handles, nested payload references or Journal schema are
introduced. Historical replay validates stored assertions and associations;
it never recompiles or rerenders old context as today's truth.
