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
source-bound scope selectors and context limit, and the Manager task machine.
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
budget include the inventory.

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
non-dispatch. Writing and the expectations step that prepares it preserve the
completion reserve; assessment may use it.
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
An adapter may send several handshake messages, one at a time, before the
session is ready or bound; the Codex App Server sends three.
Restoration preserves readiness and the first Step/send identities behind the
existing recovery barrier, without dispatching anything. Readiness cannot erase
earlier uncertainty or supply inspection handles.

`SessionChannel` implements the serialized Host `SessionIO` callbacks against
the same Coordinator and journal. Before every provider write, including tool
responses and notifications, `PrepareSend` must return a fresh acknowledged
`SendPermit` whose identity matches the actual bytes. Failed or uncertain
acknowledgement never grants dispatch. Re-observing a preparation never grants
another send, even after restart. Actual sent/not-sent/uncertain observations
remain separate. A write that answers a tool call names that call; it may follow
only the recorded Host answer to that call in the same step, and only once.
Shutdown control sends stay available during reconciliation without clearing
earlier unknown activity or charging new Agent work.

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
handler is not called. That refusal only settles the call in the journal: no
ordinary write is admitted in that state, so the adapter never sends it, and
the Host stops the session. Calls are deduplicated by session channel and provider
call identity, including the exact step, tool identity and
content/representation: an exact repeat returns the recorded answer without
executing again, and a changed call conflicts. A handler error or an invalid or
oversize outcome leaves the call pending; it cannot execute again implicitly.
If recording the answer fails, `Call` returns the error together with the
outcome, which the adapter must not send; once reconciliation shows the answer
absent, the Host records it through `ToolReconciliation`. Recorded answers are
never redacted and always carry the tool-outcome media type. A call's input is
the observed call, a JSON object that names the tool and carries its arguments
as the `arguments` member; `CallArguments` extracts them.

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
elapsed settlement; no port is executed during Restore. Under basis v3 a step
observation carries no proposed output: outcomes arrive only through answered
Host tool calls.

The consumer-owned `Workspaces` and `Checks` ports define what the repository
and check adapters implement: materialize an exact tree on a runtime volume,
harvest the complete observed tree, close it, and run one fixed check on a
disposable copy. They carry no ownership or review semantics; a check passes
only on an observed zero exit that neither timed out nor is uncertain.

The concrete Session, Workspace, Check, journal and publication adapters live
outside this package. The full contracts remain in
the design documents.

## Manager tasks

This section implements the [core protocol](../../docs/design/canonical-model/manager-apply-core.md)
for basis v3. Rule IDs refer to that document.

When a run is prepared, every Manager of the recorded topology gets one task.
`Next` returns the one admissible step of the run (T4): Explore children before
parents (T1), reviews of submissions at any time (T3), and work, integration
and the expectations steps that precede them only after every task is explored
(T2, DEC-049). Ties break with reply steps first, then by
Manager path, then by kind. The admission names the phase, the provider home
and the conversation to resume: a Manager's steps continue its conversation
(S3), while expectations and candidate steps start fresh conversations, and
the candidate step receives the expectations that were recorded before it
(T7). An incomplete
task stops all admission (T14).

`PrepareTaskSessionCommand` and `PrepareTaskStepCommand` are the only way to
start a step. The reducer checks the phase's role, tool catalogue, workspace
mode, provider home, conversation and reservation purpose, and keeps one step
at a time (S5). Step preparation compiles the original model again, matches the
whole binding and renders the phase's view: the Manager sees its own artifacts
in the step's version, its open findings and its children's states and reports;
the expectations review sees the model and the open findings; the candidate
review also sees the frozen candidate, its proposal and the recorded
expectations. The caller supplies that version and the quoted payloads that
`StepQuotes` names as `StepFacts`.

A step ends with exactly one answered outcome tool call. `CheckOutcome` checks
its arguments against the phase and the run before the Host answers it, and
`ConcludeStepCommand` records the conclusion after the container stopped,
applying the same check again. A step without a valid outcome, or one whose
S4 check failed, is failed and returns its task to the state before the step
(T6). Writing steps record their S4 candidate record as the task's own
unaccepted work.

Reviews follow T7 and T8. The expectations step names each case once, and the
candidate review assesses every recorded case by that name as evidenced,
not applicable or missing; the Host refuses a review that leaves a case out
and a pass with a case missing, and records each missing case as a blocking
finding. A failed assessment needs no cases. A pass
must resolve every open blocking finding of its purpose; it is withheld while another open blocking issue covers the task,
for an integration also one of a descendant (I3), while a producer the candidate used delivered again after the
candidate's step (D4), or, for an integration, while a child is not done (T9). The
task's next step sees why its pass was withheld. Once the write barrier is
open, a reviewer records the expectations before a task's writing step for a
purpose, unless cases for that purpose still stand, so that the Manager knows
them and the review of its submission has only the candidate step; that step is
charged as preparation and does not use the completion reserve (DEC-049).
Unless its assessment failed, a review leaves its recorded expectations standing for the task's later submissions
with the same purpose, whatever their scope. Their review skips the
expectations step while the task's open findings, other than those the reviews
against these expectations raised or resolved, and the owner's decisions are
unchanged. That step sees neither the candidate, its scope nor the aggregate,
and the Manager's steps show the standing cases for each purpose (DEC-046,
DEC-047, DEC-048). Changes requested count one rejection per purpose, and an
accepted delivery for the purpose restarts the count (DEC-050); at the bound
rejection limit the task becomes incomplete. A needs-decision verdict
blocks the task on an owner question, and a failed assessment starts a fresh
review. An integration review may address a finding to a descendant that owns
the affected artifacts (D5), with changes requested; such a blocking finding
returns the done descendant to work, blocks its ancestors below the reviewed Manager on
their children, and the reviewed Manager integrates again once they are done
(T11). Accepted deliveries advance the aggregate version when they carry
changes. Blocked tasks wake when what they named arrives (T10); a wait that only the waiting Manager itself could end, such as the delivery of a Manager that waits for it, is refused, and the root's
completion moves the run to Verifying (T13).

`Engine` drives admitted steps through the `Sessions` and `Workspaces` ports,
one at a time. It holds what the reducer does not: the compiled model and every
aggregate version. It
materializes each step's version (D3), answers the Host tools, captures the
candidate after the step (S4) and composes an accepted delivery onto the
aggregate before it concludes the review (D2). A refused composition finishes
the run Failed. A step the Host cannot finish is stopped. A confirmed stop ends
all activity in the container (S2), so its start and step are settled as
observed, an unstarted one is released, and a run that an uncertain
observation moved to Reconciling resumes; the step then concludes as failed,
its reservations and registry proposals are dropped with it, and the Manager's
next step sees a note (T6). Without a confirmed stop the effects stay for reconciliation.
A failure before the container starts costs nothing, so `Run` returns it
instead of retrying. Path reservations and registry proposals are checked when
they are made and recorded with the concluded writing step (S4), so a candidate
record carries the claims its new files used and a new engine forms the same
claims; it rebuilds the aggregate from the recorded deliveries. `Run` advances until nothing is admissible, then
leaves the run Verifying, moves it to Waiting for an owner question (W1), or
finishes it Incomplete.

Messages follow M1 to M5. A Manager sends a question, reply or hint with
`send`; `CheckSend` checks it against the tree and returns the message's
identity, and the message takes effect when the step concludes, before the
step's outcome, which may wait on it. A question to `parent` goes to the
direct parent, a question to another Manager travels along the tree, and the
Managers on its path see it in their next step without being dispatched (M2).
A question becomes an issue addressed to its addressee; a reply from that
addressee resolves it and wakes an asker waiting for it (I2, T10). A question
that none of a blocked task's Explore, work or integrate steps has seen makes
it ready, so that its next step answers it and can act on it (T10); hints and
replies it did not wait for do not wake it. A Manager that cannot otherwise
act answers in a reply step and keeps its state: a done or explored one (T11),
one awaiting the owner's decision on its frozen submission (T8), and a blocked
one whose last step saw the question but left it unanswered. Once every Manager has explored,
the reply step of a done or blocked task may correct the Manager's own files,
except while the owner decides on its frozen submission (T8); a change, even
one the Host refuses, returns the task to work with any refusal as its failure,
and a done task's done ancestors wait for it again. An integration composes
done children only: its submission and its pass wait for every child (T9). Each open question
admits one reply step: a reply step takes the question it answers, or the
oldest one when it sends no reply; a reply step the Host could not finish
drops its reply and is admitted again. A question to the owner
travels up to the root; when nothing else is admissible the run waits for the
owner (M4, W1). `AnswerCommand` records the owner's answer: it wakes the asker,
or starts a fresh review when a review asked (T8), and `ResumeRun` continues
the run. The owner's decisions about a task reach its later steps and reviews.
A parent's hints reach its direct children. A failed step's messages are
dropped with it (T6).

Not yet implemented: a parent forwarding or broadcasting a question it cannot
answer (M3, it asks anew instead), `assign_owner` and intake (T2, D6), the
dispatch of a common parent for waits nothing satisfies (T12, the run finishes
Incomplete instead), review batching, open issues carried
into a continuation run (I5), a fresh conversation
with a handoff when resume fails (S3), session rotation at `maxSessionSteps`,
an answer that requires a model change (W3), and the parent's assessment after
two rejections.

## Verification and publication

The basis binds the names of the required checks. While the run is Verifying,
`PrepareCheckCommand` admits one check as assessment work, with room for its
container's start and stop, and `ObserveCheckCommand` records its receipt and
settles its reservation; an uncertain receipt moves the run to Reconciling. A
check passes on an observed zero exit of the same image and tree.

A candidate reviewer may run a configured check on the frozen candidate with
`request_check` (`PrepareReviewCheckCommand`). The check must fit the step's
remaining time and the budget, or the call is refused and the review goes on;
its time is charged once, as the check's own invocation, and the step's
charge leaves it out. The candidate tree it runs on does not yet contain the
regenerated registry shards that publication writes.
`Engine.Verify` runs the required checks on the tree that publishing the
aggregate writes, reuses a check that already passed on it, and finishes the
run Incomplete when one fails or the budget no longer fits it.
`VerifyRunCommand` is the final evidence check:
every task is done, no blocking issue is open, nothing is running and every
required check passed on that tree; it moves the run to Publishing.

`PreparePublicationCommand` records the change set before the first write,
bound to the captured source and the verified tree, and `PublishRunCommand`
records the publisher's receipt; until then the run cannot finish. Applied requires a
non-empty change set and NoChange an empty one; stale and failed finish the run
as such, and a partial or unknown publication moves it to Reconciling. These
are the only commands that finish a run Applied or NoChange. A step's deadline
leaves its container's stop inside the step's reservation, and the adapter's
records of a step are written even after its deadline.
