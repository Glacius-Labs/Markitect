---
kind: Operation
---
# Add family

Creates the family.

- **Wire name:** `addFamily`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `familyId`: identifier
  - `name`: text
- **Refusals** at check 7, in this order:
  1. `already_exists` when a family with `familyId` exists.
- **Result:**
  - `family`: a [[equipment/family]] record
- **Permissions:** admin all.
- **Audit target:** `familyId`.
