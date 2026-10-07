# Independent review: explicit additional context allocation

Date: 2026-10-07. Separate `independent_review` subagent, read only. The parent
implemented and tested; `case_contracts` independently authored the six allocation
tests. Neither reviewer nor test author ran a model, metadata RPC or native product
binary. The reviewer ran no tests and changed no files.

Reviewed: fixed decision/path/identity and full predecessor binding, one atomic
local reservation and cumulative four-start limit, no replacement Grant/refill,
original dispatch recovery, null historical/local counters, context-only draft
preparation and source/runtime pins. Ordinary study `ledger.py` is unchanged.

Closed findings:

1. The historical reader was a critical unpinned runtime dependency. It is now in
   `dispatch.runtime_pins()` alongside the new allocation module and public
   resource decision; Protocol binds all those exact bytes.
2. Resume could initialize an empty/unbooked file before finding no prior record.
   The allocation now preflights its exact persisted binding and requested
   dispatch ID using SQLite read-only access before calling the base initializer.
   Empty/missing/unbooked recovery cannot create work or bind a new authority.

Reservation also checks the durable allocation row and current full predecessor
history immediately before booking. Historical identity, request/result hashes,
Grant allowances/exhaustion, runner argv, status and null counters are retained.
The new window uses unchanged v2 admission; unknown local tokens stay blocked.

Final reviewer conclusion: no remaining material finding in the reviewed
allocation/deduplication/history/resume scope. The public decision still records
`runGrantIssued:false`; resource authorization is distinct from the exact future
executable Grant. This package's authority documents remain drafts. Final test
and hash reconciliation results are delivered separately in the frozen handoff.
