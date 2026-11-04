---
kind: Operation
---
# Get inspection status

Shows the plan of the machine and whether it is due. `due` is `true` when the date of the request time is on or after the plan's `nextDueOn`, otherwise `false`.

- **Wire name:** `getInspectionStatus`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `machineId`: identifier, naming an existing [[equipment/machine]]
- **Refusals** at check 7, in this order:
  1. `no_plan` when the machine has no inspection plan.
- **Result:**
  - `plan`: an [[inspections/inspection-plan]] record
  - `due`: flag
- **Permissions:** staff all, admin all.
