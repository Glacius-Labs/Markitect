---
kind: Operation
---
# Create order

Creates an [[purchasing/order]] for the supplier with the next order identifier, status `draft`, `createdBy` the actor, `totalCents` 0 and no lines.

- **Wire name:** `createOrder`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `supplierId`: identifier, naming an existing [[purchasing/supplier]]
- **Refusals** at check 7, in this order:
  1. `supplier_blocked` when the supplier's status is `blocked`.
- **Result:**
  - `order`: a [[purchasing/order]] record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `orderId`.
