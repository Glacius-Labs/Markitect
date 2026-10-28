---
kind: std/Rule
---
# Audit

Operations are either mutations or reads, as each operation states. Every request to a mutation that is answered `ok` appends exactly one [[audit-entry|audit entry]], even when the operation changes no data, such as setting a member's standing to its current value. Reads, refusals and invalid requests never append an entry. Entries are never changed or removed.

An audit entry has these fields: `seq`, the entry's position in the log, starting at 1 and increasing by 1 for each entry; `at`, the request time; `actor`, the acting user's identifier; `op`, the operation name; `target`, a string that identifies the record the operation acted on, as each mutation's [[Operation|audit target]] states.

Admins read the log with [[list-audit-entries|`listAuditEntries`]].
