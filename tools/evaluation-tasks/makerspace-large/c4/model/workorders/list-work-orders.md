---
kind: Operation
---
# List work orders

Returns all work orders of the machine, open and closed, in creation order.

- **Wire name:** `listWorkOrders`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `machineId`: identifier, naming an existing [[equipment/machine]]
- **Result:**
  - `machineId`: identifier
  - `workOrders`: a list of [[workorders/work-order]] records
- **Permissions:** staff all, admin all.
