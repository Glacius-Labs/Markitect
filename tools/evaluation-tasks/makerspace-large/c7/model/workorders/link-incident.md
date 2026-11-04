---
kind: Operation
---
# Link incident

The work order's `incidentId` becomes the supplied incident. The incident may be open or resolved, and several work orders may link the same incident; a work order links only one.

- **Wire name:** `linkIncident`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `workOrderId`: identifier, naming an existing [[workorders/work-order]]
  - `incidentId`: identifier, naming an existing [[safety/incident]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the work order is closed.
  2. `incident_other_machine` when the incident's `machineId` is not the work order's `machineId`.
  3. `already_linked` when the work order's `incidentId` is not `null`.
- **Result:**
  - `workOrder`: a [[workorders/work-order]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `workOrderId`.
