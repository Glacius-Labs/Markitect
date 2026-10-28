---
kind: std/Concept
---
# Utilization row

One machine's row of the utilization report.

**Fields**, in the formats of [[values]]:

- `machineId`: identifier, naming an existing [[equipment/machine]]
- `openMinutes`: minute count
- `maintenanceMinutes`: minute count
- `availableMinutes`: minute count
- `usedMinutes`: minute count
- `utilizationPercent`: integer of at least `0` (a utilization in percent; it can exceed 100)
