---
kind: std/Concept
---
# Escalation

The record that an incident was escalated, generated identifier `esc-N`; see [[escalations/escalation-effects]]. An incident is escalated at most once.

**Fields**, in the formats of [[values]]:

- `escalationId`: identifier
- `incidentId`: identifier, naming an existing [[safety/incident]]
- `machineId`: identifier, naming an existing [[equipment/machine]] (the incident's machine)
- `workOrderId`: identifier, naming an existing [[workorders/work-order]] (the one the escalation opened)
- `cancelledReservationIds`: a list of identifiers of [[bookings/reservation]] records, empty when none was cancelled, in the order they were cancelled
- `at`: timestamp (the request time)

A record is identified by `escalationId`.
