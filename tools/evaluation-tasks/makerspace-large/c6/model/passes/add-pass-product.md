---
kind: Operation
---
# Add pass product

Creates the pass product.

- **Wire name:** `addPassProduct`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `productId`: identifier
  - `name`: text
  - `minutes`: integer from `1` to `100000` (the minutes of a pass)
  - `priceCents`: integer from `1` to `100000000` cents (the price of a pass)
  - `validDays`: integer from `1` to `3650` (the days a pass is valid)
- **Refusals** at check 7, in this order:
  1. `already_exists` when a pass product with `productId` exists.
- **Result:**
  - `product`: a [[passes/pass-product]] record
- **Permissions:** admin all.
- **Audit target:** `productId`.
