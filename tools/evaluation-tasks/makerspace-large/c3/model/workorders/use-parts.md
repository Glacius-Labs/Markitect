---
kind: Operation
---
# Use parts

The part's stock decreases by `quantity`, and the work order's `partsCostCents` grows by `quantity × unitCostCents` of the part. The `part` result is the part after the change.

- **Wire name:** `useParts`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `workOrderId`: identifier, naming an existing [[workorders/work-order]]
  - `partId`: identifier, naming an existing [[parts/part]]
  - `quantity`: integer from `1` to `1000` (the number of units used in one request)
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the work order is closed.
  2. `part_wrong_family` when the part's `familyId` is not the family of the work order's machine.
  3. `insufficient_parts` when the part's stock is lower than `quantity`.
- **Result:**
  - `workOrder`: a [[workorders/work-order]] record
  - `part`: a [[parts/part]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `workOrderId`.
