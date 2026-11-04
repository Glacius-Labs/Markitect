---
kind: Operation
---
# Receive order

Receives `quantity` units of the material against the order, as [[purchasing/receiving]] states. Ledger postings are made.

Rules of other modules that apply here: [[purchasing/receiving]], stocktake freeze and [[ledger/postings]].

- **Wire name:** `receiveOrder`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `orderId`: identifier, naming an existing [[purchasing/order]]
  - `materialId`: identifier, naming an existing [[materials/material]]
  - `quantity`: integer from `1` to `100000` (the number of units received in one request)
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the order's status is not `approved`.
  2. `not_on_order` when the order has no line for `materialId`.
  3. `over_receipt` when the line's `receivedQuantity` plus `quantity` is above the line's `quantity`.
  4. `stocktake_open` when the material has a count in an open stocktake (stocktake freeze).
- **Result:**
  - `order`: a [[purchasing/order]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `orderId`.
