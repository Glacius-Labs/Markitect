---
kind: Operation
---
# Add supplier

Creates the [[purchasing/supplier]] with status `active`.

- **Wire name:** `addSupplier`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `supplierId`: identifier
  - `name`: text
- **Refusals** at check 7, in this order:
  1. `already_exists` when a supplier with `supplierId` exists.
- **Result:**
  - `supplier`: a [[purchasing/supplier]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `supplierId`.
