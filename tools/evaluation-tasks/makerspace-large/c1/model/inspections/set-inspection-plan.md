---
kind: Operation
---
# Set inspection plan

Sets the inspection interval of the machine. A machine without a plan gets a new plan: `lastPassedOn` is the date of the request time and `failed` is `false`. A machine with a plan keeps its `lastPassedOn` and its `failed`. In both cases `intervalDays` is the supplied value and `nextDueOn` is computed from it ([[inspections/inspection-cycle]]). Setting the current interval succeeds without a change. Lockouts are not touched.

- **Wire name:** `setInspectionPlan`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `machineId`: identifier, naming an existing [[equipment/machine]]
  - `intervalDays`: integer from `1` to `3650` (the days between inspections)
- **Result:**
  - `plan`: an [[inspections/inspection-plan]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `machineId`.
