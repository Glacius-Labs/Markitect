---
kind: Operation
---
# List consumption

Returns the rows of the machine in `materialId` ascending order.

- **Wire name:** `listConsumption`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `machineId`: identifier, naming an existing [[equipment/machine]]
- **Result:**
  - `machineId`: identifier
  - `rows`: a list of [[consumables/consumption-row]] records
- **Permissions:** staff all, admin all.
