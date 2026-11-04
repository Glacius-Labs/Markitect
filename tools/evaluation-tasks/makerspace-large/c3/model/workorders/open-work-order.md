---
kind: Operation
---
# Open work order

Creates a work order with the next work order identifier, `status` `open`, `incidentId` `null`, `partsCostCents` 0, `openedBy` the actor, `openedAt` the request time, and `closedAt` and `note` `null`. It applies a lockout to the machine with source `workOrder`, `reason` the supplied `summary`, `appliedBy` the actor and `appliedAt` the request time, and the work order's `lockoutId` is that lockout's identifier. The machine's other lockouts stay.

Rules of other modules that apply here: [[lockouts/lockout-gate]].

- **Wire name:** `openWorkOrder`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `machineId`: identifier, naming an existing [[equipment/machine]]
  - `summary`: text
- **Result:**
  - `workOrder`: a [[workorders/work-order]] record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `workOrderId`.
