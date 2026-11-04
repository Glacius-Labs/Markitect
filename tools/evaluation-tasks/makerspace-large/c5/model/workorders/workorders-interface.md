---
kind: Interface
---
# Workorders interface

What other modules may use of workorders: its operations and the work order record. Other modules also ask the capability listed below.

- **Operations:** [[workorders/open-work-order]], [[workorders/link-incident]], [[workorders/use-parts]], [[workorders/close-work-order]], [[workorders/list-work-orders]].
- **Uses:** [[members/members-interface]], [[equipment/equipment-interface]], [[safety/safety-interface]], [[parts/parts-interface]], [[lockouts/lockouts-interface]].
- **Records:** [[workorders/work-order]].

Capability other modules ask of workorders:

- Opening a work order for a machine with a summary, for a given opener and already linked to a given incident of that machine. It has the effects of [[workorders/open-work-order]] followed by [[workorders/link-incident]], without their refusals and without an audit entry of its own, and it returns the work order. Escalations uses this.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_linked` | The work order is already linked to an incident. |
| `incident_open` | The incident linked to the work order is not resolved. |
| `incident_other_machine` | The incident is about another machine than the work order's. |
| `insufficient_parts` | The part's stock is lower than the requested quantity. |
| `part_wrong_family` | The part fits another family than the work order's machine. |
