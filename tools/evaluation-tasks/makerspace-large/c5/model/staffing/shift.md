---
kind: std/Concept
---
# Shift

A period in which a staff user or an admin is on duty, generated identifier `shf-N`. A shift is never removed; a cancelled shift counts for nothing.

**Fields**, in the formats of [[values]]:

- `shiftId`: identifier
- `userId`: identifier, naming an existing [[members/user]]
- `start`: timestamp
- `end`: timestamp
- `status`: `scheduled` or `cancelled`

A record is identified by `shiftId`.
