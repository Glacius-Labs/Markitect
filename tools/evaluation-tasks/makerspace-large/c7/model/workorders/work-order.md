---
kind: std/Concept
---
# Work order

A repair job on one machine, generated identifier `wko-N`; see [[workorders/work-order-lifecycle]]. `incidentId` is `null` until an incident is linked. `closedAt` and `note` are `null` while the work order is open.

**Fields**, in the formats of [[values]]:

- `workOrderId`: identifier
- `machineId`: identifier, naming an existing [[equipment/machine]]
- `summary`: text
- `status`: `open` or `closed`
- `incidentId`: identifier, naming an existing [[safety/incident]], or `null`
- `lockoutId`: identifier, naming an existing [[lockouts/lockout]]
- `partsCostCents`: cent amount
- `openedBy`: identifier, naming an existing [[members/user]]
- `openedAt`: timestamp
- `closedAt`: timestamp, or `null`
- `note`: text, or `null`

A record is identified by `workOrderId`.
