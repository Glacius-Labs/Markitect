---
kind: Operation
---
# Restock material

The stock increases by `quantity`.

Rules of other modules that apply here: stocktake freeze.

- **Wire name:** `restockMaterial`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `materialId`: identifier, naming an existing [[materials/material]]
  - `quantity`: integer from `1` to `100000` (the number of units added to stock in one request)
- **Refusals** at check 7, in this order:
  1. `stocktake_open` when the material has a count in an open stocktake (stocktake freeze).
- **Result:**
  - `material`: a [[materials/material]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `materialId`.
