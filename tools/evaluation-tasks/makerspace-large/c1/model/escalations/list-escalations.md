---
kind: Operation
---
# List escalations

Returns all escalations in creation order.

- **Wire name:** `listEscalations`, a read.
- **Arguments:** only `actor` and `at`.
- **Result:**
  - `escalations`: a list of [[escalations/escalation]] records
- **Permissions:** staff all, admin all.
