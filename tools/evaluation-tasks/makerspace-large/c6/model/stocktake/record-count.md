---
kind: Operation
---
# Record count

Records the [[stocktake/count]] of the material in the stocktake. When the stocktake already has a count of that material, the new `counted` replaces it. The material's stock does not change until the stocktake is closed. From the moment a material has a count in an open stocktake, the [[stocktake/stocktake-freeze]] applies to it.

- **Wire name:** `recordCount`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `stocktakeId`: identifier, naming an existing [[stocktake/stocktake]]
  - `materialId`: identifier, naming an existing [[materials/material]]
  - `counted`: integer from `0` to `100000000` (the number of units counted)
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the stocktake is closed.
- **Result:**
  - `stocktake`: a [[stocktake/stocktake]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `stocktakeId/materialId`.
