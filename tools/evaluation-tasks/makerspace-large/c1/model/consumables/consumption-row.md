---
kind: std/Concept
---
# Consumption row

How fast one machine uses up one material while it runs; see [[consumables/consumption]]. A machine has at most one row per material.

**Fields**, in the formats of [[values]]:

- `machineId`: identifier, naming an existing [[equipment/machine]]
- `materialId`: identifier, naming an existing [[materials/material]]
- `unitsPerHour`: integer from `1` to `1000` (the units of the material used in one hour)

A record is identified by `machineId` and `materialId` together.
