---
kind: Operation
---
# Set supplier status

Sets the supplier's status. Setting the current status succeeds without a change. Orders that exist are not changed; the status is judged when an order is created and when an order is approved.

- **Wire name:** `setSupplierStatus`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `supplierId`: identifier, naming an existing [[purchasing/supplier]]
  - `status`: a [[purchasing/supplier-status]]: `active` or `blocked`
- **Result:**
  - `supplier`: a [[purchasing/supplier]] record
- **Permissions:** admin all.
- **Audit target:** `supplierId`.
