---
kind: Operation
---
# Add order line

Adds a line to the order, after its other lines, with `receivedQuantity` 0. The order's `totalCents` increases by `quantity × unitCostCents`; this product is not discounted and not rounded. The supplier's status is not judged.

- **Wire name:** `addOrderLine`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `orderId`: identifier, naming an existing [[purchasing/order]]
  - `materialId`: identifier, naming an existing [[materials/material]]
  - `quantity`: integer from `1` to `100000` (the number of units ordered)
  - `unitCostCents`: integer from `1` to `1000000` cents (the cost of one unit from the supplier)
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the order's status is not `draft`.
  2. `duplicate_line` when the order already has a line for `materialId`.
- **Result:**
  - `order`: a [[purchasing/order]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `orderId`.
