---
kind: Operation
---
# Add PPE item

Creates the item with the given stock.

- **Wire name:** `addPpeItem`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `itemId`: identifier
  - `name`: text
  - `stock`: integer from `0` to `100000` (the number of units of an item in stock)
- **Refusals** at check 7, in this order:
  1. `already_exists` when an item with `itemId` exists.
- **Result:**
  - `item`: a [[ppe/ppe-item]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `itemId`.
