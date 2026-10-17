# SQLite journal and repository control

`New()` returns the concrete `*Control` implementation of `apply.Control`.
`Acquire` accepts the producer's `ControlRequest` and returns a session whose
`Journal` implements the four `run.Journal` operations. It interprets neither
the opaque publication guard nor the model. Source recapture and restoration
belong to `apply.ControlService` and `run.Restore`.

The repository must have `.git` and `.markitect` metadata. A private
`.markitect/state/` directory holds the process-lifetime `journal.lock` and the
checked `journal.json` store registration. The selected external state directory
is dedicated to `journal.sqlite` and its SQLite sidecars. Its parent must already
exist. Both directories must be private to the current OS user; unknown existing
directories are checked, never repaired. Redirected paths and hardlinked files
are refused. Linux checks ownership and permissions; Windows creates a protected,
inheritable owner-only DACL and validates actual ownership and allowed ACEs.
Other lock backends fail closed.

Initialization records an `initializing` registration before creating a database,
then writes and flushes the schema and completes the registration. Interrupted
initialization needs reconciliation. A persistent lock without its registration
also needs reconciliation even when another empty state directory is selected.
A registered missing, copied or replaced
database, another checkout, another Host identity assertion or another selected
state directory cannot silently start empty history. Store identity is random;
repository, registration and database physical identities are also recorded.
Registration is finalized in place to preserve its checked-created identity.
Linux flushes newly created directory entries in their parents before proceeding;
Windows uses file flushing and the host filesystem's directory guarantees.
Each acquisition
increments the signed 63-bit controller epoch; exhaustion refuses acquisition.
This package supplies no administrative reconciliation or migration command.

Each operation uses one bounded, dedicated `database/sql` connection to pinned
`modernc.org/sqlite`. WAL, FULL synchronization, foreign keys and the 250 ms busy
timeout are verified on that actual connection. A context bounds admission and
SQL work to five seconds. A session serializes operations; `BEGIN IMMEDIATE`
holds SQLite's writer lock from the live-epoch check through the operation's
commit. No transaction waits for an Agent, check or other external effect.

Filesystem checks before and after transactions detect persistent control or
database replacement and poison the handle. They are not an atomic filesystem
sandbox against another process running as the same OS user. That user can also
rewrite a database in place or restore an older backup over the same inode;
physical identity does not authenticate history or detect every such rollback.
The private local filesystem and cooperative lifetime lock are the supported
boundary. Use a local filesystem with SQLite locking and synchronization support;
network filesystems are outside this adapter's guarantee.

Blobs share immutable bytes by SHA-256. Reads reconstruct the exact requested
digest, length, media type and encoding; identical bytes may have several valid
interpretations. Append uses RUN's canonical codec, checks batch identity before
revision CAS, rejects reused event IDs, and verifies the required basis and owner
payloads through `BindingRefs` and `BindingOwnerRef`. All rows and the head change
in one transaction. `SessionRefs` supplies the required stored profile, tool,
step, permitted wire representation and result references. A wire identity is
not a stored payload; redacted journal bytes never imply exact replayable output.
No adapter reducer decides event meaning.

Reads pin a committed head and return whole batches at exact boundaries, limited
to 256 batches and 64 MiB of canonical batch bytes per page. `Next` continues the
same pinned interval. Exact batch lookup reports the current head and observed
presence or absence. Length, canonical bytes, digest, sequence and event indexes
are checked; malformed or missing required payloads are errors.

Returned references, receipts and pages remain observations beside an error.
Only a successful exact receipt acknowledges commit. An unconfirmed commit or
identity loss cannot authorize a dependent external effect; retain the known
identity and reconcile through a fresh controlled handle. Closing control or
acquiring a new epoch never proves that previous processes stopped. Process-kill
tests establish only the observed process-loss/reopen behavior, not power-loss
durability, provider execution, public Apply readiness or human acceptance.
