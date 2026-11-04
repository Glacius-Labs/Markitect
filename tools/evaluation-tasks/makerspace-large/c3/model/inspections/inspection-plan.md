---
kind: std/Concept
---
# Inspection plan

The inspection plan of one machine; see [[inspections/inspection-cycle]] for how its dates move.

**Fields**, in the formats of [[values]]:

- `machineId`: identifier, naming an existing [[equipment/machine]]
- `intervalDays`: integer from `1` to `3650` (the days between inspections)
- `lastPassedOn`: date
- `nextDueOn`: date, `intervalDays` days after `lastPassedOn`
- `failed`: flag, `true` when the latest inspection failed

A record is identified by `machineId`.
