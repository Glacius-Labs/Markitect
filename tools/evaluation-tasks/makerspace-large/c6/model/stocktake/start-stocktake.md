---
kind: Operation
---
# Start stocktake

Creates a [[stocktake/stocktake]] with the next stocktake identifier, status `open` and no counts.

- **Wire name:** `startStocktake`, a mutation.
- **Arguments:** only `actor` and `at`.
- **Refusals** at check 7, in this order:
  1. `stocktake_running` when a stocktake with status `open` exists.
- **Result:**
  - `stocktake`: a [[stocktake/stocktake]] record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `stocktakeId`.
