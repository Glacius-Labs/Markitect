---
kind: Operation
---
# Require PPE

Makes the item a requirement of the family. Repeating a requirement succeeds without a change.

- **Wire name:** `requirePpe`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `familyId`: identifier, naming an existing [[equipment/family]]
  - `itemId`: identifier, naming an existing [[ppe/ppe-item]]
- **Result:**
  - `requirement`: a [[ppe/ppe-requirement]] record
- **Permissions:** admin all.
- **Audit target:** `familyId/itemId`.
