---
kind: Operation
---
# Close work order

The work order gets status `closed`, `closedAt` the request time and `note` as supplied. The lockout named by its `lockoutId` is released: it gets `releasedAt` the request time. Other lockouts of the machine stay.

Rules of other modules that apply here: [[lockouts/lockout-gate]].

- **Wire name:** `closeWorkOrder`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `workOrderId`: identifier, naming an existing [[workorders/work-order]]
  - `note`: text
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the work order is already closed.
  2. `incident_open` when the work order has a linked incident whose `status` is `open`.
- **Result:**
  - `workOrder`: a [[workorders/work-order]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `workOrderId`.
