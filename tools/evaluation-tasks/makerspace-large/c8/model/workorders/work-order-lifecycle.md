---
kind: std/Rule
internal: true
---
# Work order lifecycle

A [[workorders/work-order]] is a repair job on one machine. Opening it applies a lockout of source `workOrder` to the machine ([[lockouts/lockouts]]), and the work order keeps that lockout's identifier as `lockoutId`. The machine stays locked out by this lockout until the work order is closed; closing releases exactly that lockout. Other active lockouts of the machine are not touched, so a machine with a manual lockout stays locked out after its work order closes.

While a work order is `open`, it can be linked to one incident of its machine, and spare parts of the machine's family can be used on it. A used part leaves the part's stock and adds quantity times the part's `unitCostCents` to the work order's `partsCostCents`. The cost is only recorded on the work order; no charge is created for anyone.

A work order cannot be closed while its linked incident is `open`. A work order without a linked incident can be closed at once. A closed work order cannot be changed, and work orders are never removed.

Work orders are opened by staff or an admin, or by the escalations module for an escalated incident.
