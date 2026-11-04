---
kind: Interface
---
# Inspections interface

What other modules may use of inspections: its operations, the inspection plan record, and the rule that a machine whose inspection is due cannot be reserved. Other modules also ask the capabilities listed below.

- **Operations:** [[inspections/set-inspection-plan]], [[inspections/record-inspection]], [[inspections/get-inspection-status]], [[inspections/list-inspections-due]].
- **Uses:** [[equipment/equipment-interface]], [[lockouts/lockouts-interface]].
- **Records:** [[inspections/inspection-plan]].
- **Rules that bind other modules:** [[inspections/inspection-gate]].

Capabilities other modules ask of inspections:

- The inspection plan of a machine, or that it has none. Compliance asks this.
- Whether a machine is due on a date, which is whether it has a plan and the date is on or after the plan's `nextDueOn`. Bookings and compliance ask this.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `inspection_due` | The machine has an inspection plan and is due on the relevant date. |
| `no_plan` | The machine has no inspection plan. |
