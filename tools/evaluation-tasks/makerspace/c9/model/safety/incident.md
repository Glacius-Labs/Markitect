---
kind: std/Concept
---
# Incident

A safety incident recorded against a machine, generated identifier `inc-N`. `resolution` and `resolvedAt` are `null` while open.

**Fields**, in the formats of [[values]]:

- `incidentId`: identifier
- `machineId`: identifier, naming an existing [[equipment/machine]]
- `severity`: one of `low`, `medium` or `high`
- `description`: text
- `reportedBy`: identifier
- `reportedAt`: timestamp
- `status`: `open` or `resolved`
- `resolution`: text, or `null`
- `resolvedAt`: timestamp, or `null`

A record is identified by `incidentId`.
