---
kind: std/Concept
---
# Audit entry

One entry of the audit log. Entries are never changed or removed. `target` is `null` in a refusal entry.

**Fields**, in the formats of [[values]]:

- `seq`: count
- `at`: timestamp
- `actor`: identifier
- `actorRole`: one of `member`, `staff`, `admin` or `steward`; see authorization
- `op`: operation name
- `target`: audit target, or `null`
- `outcome`: `accepted` or `refused`
