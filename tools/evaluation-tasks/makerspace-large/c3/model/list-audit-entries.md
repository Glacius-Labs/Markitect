---
kind: Operation
---
# List audit entries

Returns all audit entries in `seq` order.

- **Wire name:** `listAuditEntries`, a read.
- **Arguments:** only `actor` and `at`.
- **Result:**
  - `entries`: a list of [[audit-entry]] records
- **Permissions:** admin all.
