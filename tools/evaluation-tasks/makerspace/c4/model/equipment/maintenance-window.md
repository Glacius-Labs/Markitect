---
kind: std/Concept
---
# Maintenance window

A maintenance window of a machine, generated identifier `mnt-N`.

**Fields**, in the formats of [[values]]:

- `windowId`: identifier
- `machineId`: identifier, naming an existing [[equipment/machine]]
- `start`: timestamp
- `end`: timestamp
- `reason`: text
- `status`: `scheduled` or `cancelled`

A record is identified by `windowId`.
