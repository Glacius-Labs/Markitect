---
kind: Operation
---
# Clear consumption

Removes the row of the machine and the material. Charges and stock draws made earlier stay as they are.

- **Wire name:** `clearConsumption`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `machineId`: identifier, naming an existing [[equipment/machine]]
  - `materialId`: identifier, naming an existing [[materials/material]]
- **Refusals** at check 7, in this order:
  1. `no_consumption` when the machine has no row for the material.
- **Result:**
  - `machineId`: identifier
  - `materialId`: identifier
- **Permissions:** staff all, admin all.
- **Audit target:** `machineId/materialId`.
