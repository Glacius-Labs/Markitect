---
kind: Operation
---
# List inspections due

Returns the plans whose `nextDueOn` is on or before `date`, in `machineId` ascending order. A plan whose `nextDueOn` equals `date` is included. Machines without a plan are not listed.

- **Wire name:** `listInspectionsDue`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `date`: date
- **Result:**
  - `date`: date
  - `plans`: a list of [[inspections/inspection-plan]] records
- **Permissions:** staff all, admin all.
