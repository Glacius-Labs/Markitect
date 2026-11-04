---
kind: Operation
---
# List orders

Returns all orders of the supplier, in creation order, in every status.

- **Wire name:** `listOrders`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `supplierId`: identifier, naming an existing [[purchasing/supplier]]
- **Result:**
  - `supplierId`: identifier
  - `orders`: a list of [[purchasing/order]] records
- **Permissions:** staff all, admin all.
