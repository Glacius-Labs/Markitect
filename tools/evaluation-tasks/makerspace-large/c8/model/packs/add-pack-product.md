---
kind: Operation
---
# Add pack product

Creates the pack product.

- **Wire name:** `addPackProduct`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `productId`: identifier
  - `name`: text
  - `minutes`: integer from `1` to `100000` (the minutes of a pack)
  - `priceCents`: integer from `1` to `100000000` cents (the price of a pack)
  - `validDays`: integer from `1` to `3650` (the days a pack is valid)
- **Refusals** at check 7, in this order:
  1. `already_exists` when a pack product with `productId` exists.
- **Result:**
  - `product`: a [[packs/pack-product]] record
- **Permissions:** admin all.
- **Audit target:** `productId`.
