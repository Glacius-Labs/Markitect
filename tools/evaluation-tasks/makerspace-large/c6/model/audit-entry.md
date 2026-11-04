---
kind: std/Concept
---
# Audit entry

One entry of the audit log. Entries are never changed or removed.

**Fields**, in the formats of [[values]]:

- `seq`: count
- `at`: timestamp
- `actor`: identifier
- `op`: operation name
- `target`: audit target
