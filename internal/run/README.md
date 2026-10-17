# Run admission

This package owns pure decisions over a fixed Apply basis and one assignment's
shared ledger. It consumes the existing model, snapshot, organization and
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

This early slice starts one assignment on an actually new journal stream and
supports changed-basis continuation with the same policy and consumed ledger.
The original owner request remains fixed for that assignment. Each Prepared run
needs an available run unit and the new basis's structural review floor in the
remaining shared ledger; this need not replenish the configured reserve that
assessment work may spend.
Assignment extensions/closure, multiple ordinary assignments, review and task
execution, final evidence, Verifying/Publishing transitions and Applied/NoChange
are not implemented yet. There is no Session, Workspace, Check, SQLite, provider
or Apply adapter here. The full contracts remain in the design documents.
