---
kind: Operation
---
# Close stocktake

Closes the stocktake: its status becomes `closed`. For each count, in ascending `materialId` order, `difference` is `counted` minus the material's stock at the request time, and the material's stock becomes `counted`. The freeze of the counted materials ends.

Ledger postings are made, for each count in that order and one per count whose `difference` is not 0, as [[ledger/postings]] states. The `units` of a posting are the absolute value of `difference` and the price is the material's current unit price at the request time. A negative `difference` is a shrink; a positive `difference` is a gain. A count with `difference` 0 posts nothing.

Example: the stock of a material is 10 and its unit price is 250 cents. Counted 8: `difference` is −2, the stock becomes 8 and the shrink posts 500 cents. Counted 15: `difference` is 5, the stock becomes 15 and the gain posts 1250 cents.

Rules of other modules that apply here: [[stocktake/stocktake-freeze]] and [[ledger/postings]].

- **Wire name:** `closeStocktake`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `stocktakeId`: identifier, naming an existing [[stocktake/stocktake]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the stocktake is closed.
  2. `nothing_counted` when the stocktake has no count.
- **Result:**
  - `stocktake`: a [[stocktake/stocktake]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `stocktakeId`.
