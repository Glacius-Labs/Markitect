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

This package supplies the executable consumer contract and controlled tests. A
physical Control/Journal adapter, SQL durability, execution and guarded publication
are still absent. Passing source checks establishes none of those capabilities.
