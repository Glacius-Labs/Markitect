---
kind: std/Concept
---
# Lockout

A lockout of one machine, generated identifier `lck-N`. It is active while `releasedAt` is `null`; see [[lockouts/lockouts]].

**Fields**, in the formats of [[values]]:

- `lockoutId`: identifier
- `machineId`: identifier, naming an existing [[equipment/machine]]
- `source`: a [[lockouts/lockout-source]]: `manual`, `inspection` or `workOrder`
- `reason`: text
- `appliedBy`: identifier (the user who applied it)
- `appliedAt`: timestamp
- `releasedAt`: timestamp, or `null`

A record is identified by `lockoutId`.
