---
kind: Operation
---
# Set consumption

Sets the row of the machine and the material. A machine with no row for the material gets one; a machine with a row has it replaced, so `unitsPerHour` is the supplied value. Setting the current value succeeds without a change.

Rules of other modules that apply here: [[consumables/consumption]].

- **Wire name:** `setConsumption`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `machineId`: identifier, naming an existing [[equipment/machine]]
  - `materialId`: identifier, naming an existing [[materials/material]]
  - `unitsPerHour`: integer from `1` to `1000` (the units of the material used in one hour)
- **Result:**
  - `row`: a [[consumables/consumption-row]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `machineId/materialId`.
