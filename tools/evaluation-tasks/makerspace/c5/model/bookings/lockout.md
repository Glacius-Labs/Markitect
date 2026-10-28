---
kind: std/Concept
---
# Lockout

A lockout of one incident's machine, generated identifier `lock-N`.

**Fields**, in the formats of [[values]]:

- `lockoutId`: identifier
- `incidentId`: identifier, naming an existing [[safety/incident]]
- `machineId`: identifier, naming an existing [[equipment/machine]]
- `start`: timestamp
- `end`: timestamp
- `cancelledReservationIds`: a list of values, each identifier, naming an existing [[bookings/reservation]]

A record is identified by `lockoutId`.
