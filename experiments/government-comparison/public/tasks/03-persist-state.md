# Task 3 — Persist order and inventory state

Persist the catalog, orders, canonical idempotency keys, and inventory reservation state in the local SQLite database supplied by the runner. Restarting the process with the same database location preserves the logical results of catalog, inventory, order reads, and idempotent retries. Schema initialization must work on a fresh database.

A failed order/reservation operation must not leave a partial success. No external database service is allowed. Add a repeatable restart check to the run instructions and focused checks. Preserve all released behavior and any existing Brownfield routes and client-visible behavior in the supplied candidate. A migration framework is not required.
