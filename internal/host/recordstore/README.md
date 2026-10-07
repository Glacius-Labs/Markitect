# Projection record store

`recordstore` is a Host-owned persistence boundary for the existing
`records.ProjectionRecord` and `records.VerificationResult` values. It keeps an
immutable source-bound event history and a separate explicit active-owner set.
It does not authenticate callers, accept semantic intent, execute checks, or
equate active ownership with verified convergence.

Host supplies an absolute external store path and every forbidden root when it
calls `Initialize` or `Open`. The store refuses overlap with source, Git
metadata, or materialization targets. Initialization creates an absent root
exclusively. It never creates parent directories, rolls back partial work, or
repairs an interrupted store.

The store contains a versioned `store.json` marker and `events/` directory.
Events are canonical closed JSON with a sequence, previous-event digest,
operation payload, and content digest. Event filenames encode sequence and
digest. The chain starts at the marker digest. Attempt events preserve complete,
partial-failure, and escalated ProjectionRecords; verification events preserve
passed, failed, and incomplete results. Active-selection events contain the
complete sorted set of active record IDs. An empty set is explicit.

Writers create an exclusive lock and stage one event beside the event log before
renaming it into its final digest-bound name. Operations compare the caller's
expected event head while holding the lock. A stale head is returned as
`ErrStaleHead`; callers must read and review current state before retrying.
Repeated record/result content IDs and an unchanged current selection are
idempotent. If a process stops with a lock or `.pending-*` event, reads and
writes report the exact path. The owner must inspect and recover it. The store
never deletes lock, pending, unknown, malformed, or partial files.

Only `materialized-unverified` records can enter the active set. Verification
results do not auto-promote or remove records: a complete materialization can
remain the selected owner even when its latest declared checks failed. A
partial or escalated attempt remains in history and cannot replace the prior
active record automatically. Host must check a result against current declared
verifier/check identities before appending it; the store validates its
content-ID and source/model/target relationship to the stored record.

`Read` performs a no-write observation: it refuses an existing writer lock or pending event, validates the full chain, then checks that no writer appeared. A concurrent append may leave a valid prior head; callers bind any following write with expected-head compare-and-swap. It validates canonical JSON encoding, unique record/result IDs, earlier same-Projection prior links, verification references, active selection references, portable artifact ownership collisions, and bounded file and event counts. Symlinks, Windows reparse points, path aliases, unknown
entries, and malformed names are refused. Digests prove byte identity and
linkage only; they do not establish authorization, semantic adequacy, verifier
independence, acceptance, or external filesystem isolation.

Root and events-directory operations now use identity-checked opened handles. After event publication, validated readback must match the exact published sequence and event digest. Otherwise the operation returns `ErrCommittedButUnobserved`, preserves partial state, and performs no rollback or retry. This is byte/history consistency, not authentication or an OS sandbox. The [Host write identity contract](../../../docs/design/host-write-identity.md) explains the object-versus-path and partial-publication boundary.
