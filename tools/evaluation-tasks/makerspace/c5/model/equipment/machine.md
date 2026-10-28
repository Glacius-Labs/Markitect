---
kind: std/Concept
---
# Machine

A machine of one family with a rate in cents per minute.

**Fields**, in the formats of [[values]]:

- `machineId`: identifier
- `name`: text
- `familyId`: identifier, naming an existing [[equipment/family]]
- `rateCentsPerMinute`: integer from `1` to `10000` cents per minute (a machine's rate per minute)

A record is identified by `machineId`.
