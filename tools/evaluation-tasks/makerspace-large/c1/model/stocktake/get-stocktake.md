---
kind: Operation
---
# Get stocktake

Returns the stocktake with its counts, in ascending `materialId` order.

- **Wire name:** `getStocktake`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `stocktakeId`: identifier, naming an existing [[stocktake/stocktake]]
- **Result:**
  - `stocktake`: a [[stocktake/stocktake]] record
- **Permissions:** staff all, admin all.
