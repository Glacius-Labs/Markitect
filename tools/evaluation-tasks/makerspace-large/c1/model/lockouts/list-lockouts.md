---
kind: Operation
---
# List lockouts

Returns all lockouts of the machine, active and released, in creation order.

- **Wire name:** `listLockouts`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `machineId`: identifier, naming an existing [[equipment/machine]]
- **Result:**
  - `machineId`: identifier
  - `lockouts`: a list of [[lockouts/lockout]] records
- **Permissions:** staff all, admin all.
