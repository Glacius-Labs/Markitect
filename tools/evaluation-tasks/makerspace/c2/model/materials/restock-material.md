---
kind: Operation
---
# Restock material

The stock increases by `quantity`.

- **Wire name:** `restockMaterial`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `materialId`: identifier, naming an existing [[materials/material]]
  - `quantity`: integer from `1` to `100000` (the number of units added to stock in one request)
- **Result:**
  - `material`: a [[materials/material]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `materialId`.
