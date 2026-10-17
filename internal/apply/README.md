# Apply preparation and controlled state

Apply owns repository/profile preparation and consumer ports. Complete committed
settings preparation can bind original captured facts through `BindRun` without
effects. Preview results have no binding or Control entrypoint. Public Apply
remains unavailable.

`ControlService.Open` consumes that successful committed result and explicit Host
facts. It acquires `Control`, validates the returned repository/store/epoch, and
recaptures the exact final selection under the lock. Actual HEAD, raw inputs,
locations, bytes and provenance must match. The original opaque publication guard
stays in the basis; a new equivalent capture may return a different guard.

The service calls bounded `run.Restore` over the same journal. A failed restore
returns no controlled projection; it never falls back to an empty stream.
Unresolved holds retain RUN's `RecoveryRequired` barrier. Opening appends nothing
and starts no run, session, check, workspace or publication.

The returned `ControlledRun` owns the session lifetime, original basis, restored
Coordinator and a journal wrapper. Host callers serialize their use. Every journal
operation checks the live binding first; lost control permanently invalidates the
handle. The physical adapter must also fence operations at their actual boundary.
Journal partial receipts and uncertain errors pass through to RUN unchanged.
`Close` is idempotent, retains its cleanup error and invalidates the handle even
when release fails. Close makes no claim that external processes stopped. A
partial acquisition is closed even on cancellation; failed recapture observations
remain separate from the original successful input.

The [physical journal contract](../../docs/design/canonical-model/manager-apply-journal.md#physical-store-and-commit-contract)
requires one repository metadata lock independent of the selected external state
directory, a checked active store registration, rejection of missing/replaced
registered databases, and a new controller epoch. Repository identity is an
explicit Host assertion until a physical adapter grounds it against the selected
acquisition location. Store identity is durably assigned and registered by that
adapter; paths and source hashes cannot invent it.

This package supplies the executable consumer contract and controlled tests.
The physical Control/Journal, session, workspace, check and publication adapters
live outside it. Passing source checks establishes none of their capabilities.

## Execution

`ControlledRun.Execute` runs the bound basis through the core with the
adapters in `ExecutionPorts`, which the Host composes: one `run.Sessions` for
both roles, `run.Workspaces`, `run.Checks`, a `Publisher`, the session profile
of each Manager's role, the request of each configured check, a clock and the
engine limits. `RoleProfile` returns the prepared Agent profile a role runs
with, from which the Host builds its adapter's session profile. Apply chooses no
provider, credential or image.

When no run is active, Execute prepares and starts one. An active run bound to
another capture finishes Stale first. The first run opens an assignment; after
an incomplete, stale or failed run, a run on a changed basis continues the
assignment; after Applied or NoChange the assignment is closed and a new one
opens. A finished basis is not run again. The Managers then work until the root
is done (`run.Engine.Run`).

In Verifying, Execute builds the publication target: the aggregate's files plus
every registry shard whose records changed, regenerated from the aggregate's
registry. A regenerated shard keeps the captured order of its remaining records
and lists new ones after them. Every configured check with `requiredBy`, which
the basis binds, runs on that exact target; a failing check finishes the run
Incomplete. The publisher then preflights the target's change set against the
captured source: a source that changed finishes the run Stale, a change set
the target cannot represent (`ErrUnpublishable`) finishes it Failed, and any
other error leaves the run Verifying for the next Execute. The final evidence
check binds the target and moves the run to Publishing. The change set is
journaled with its source and target before the first write, and the
publication receipt finishes the run Applied, NoChange, Stale or Failed, or
leaves it to reconciliation when the write was partial or its result is
unknown. A publication that an earlier process prepared is inspected, not
written again.

Execute returns when the run finished, waits for the owner or needs
reconciliation. `ExecutionResult` lists the open questions to the owner among
its issues; `Answer` records the owner's answer, and the next Execute resumes
the waiting run.
