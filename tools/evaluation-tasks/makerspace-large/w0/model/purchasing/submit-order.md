---
kind: Operation
---
# Submit order

The order's status becomes `submitted`. Lines can no longer be added.

- **Wire name:** `submitOrder`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `orderId`: identifier, naming an existing [[purchasing/order]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the order's status is not `draft`.
  2. `empty_order` when the order has no line.
- **Result:**
  - `order`: a [[purchasing/order]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `orderId`.
