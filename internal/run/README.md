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
termination to Stop. It names no engine or provider protocol. Session and step
preparation commit their finite admission reservations atomically with the
preallocated identities, fixed profile/image/workspace and input/contract
references. The original payloads must be available and verified. Reviewer
workspaces are read-only. Credential fields are references only.
Each Step explicitly asserts the still-current Session plan binding, including
profile, image, workspace and candidate. A changed or unprovable binding needs a
new session. Physical workspace currentness remains the Host's responsibility.

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

Tool ingress is deduplicated by session channel and provider call identity,
including the exact step, tool identity and content/representation. A changed
call conflicts. An unresolved duplicate stays pending and cannot execute again
implicitly. A recorded reply is returned with its original receipt and bytes
when the retained representation permits it; redaction or unavailable bytes
returns the known receipt and an observation limit instead. A fresh ingress
receipt is not tool-execution authority. Tool interpretation and actual execution
are not implemented in this slice.

Provider session/step IDs remain distinct from Host IDs. Inspect can select only
the exact journal-bound handles and never starts a container or resends. Stop
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
