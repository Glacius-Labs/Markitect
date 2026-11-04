---
kind: std/Rule
---
# Escalation effects

An escalation ([[escalations/escalate-incident]]) acts on a [[safety/incident]] that a policy says is serious enough. All effects take place together, in this order:

1. A [[workorders/work-order]] is opened for the incident's machine with the summary `Escalated incident <incidentId>`, for example `Escalated incident inc-3`. It is linked to the incident, it holds its own lockout of source `workOrder`, and its opener is the actor, all as [[workorders/open-work-order]] and [[workorders/link-incident]] state.
2. Every reservation of the machine whose reported status is `reserved` and whose `start` is after the request time is cancelled. They are taken in ascending `start` order, then in creation order. Each gets status `cancelled`, as [[bookings/cancel-reservation]] states: no fee and no credit. Its member gets a notice of kind `cancellation` and `ref` the reservation identifier, posted as [[notices/posting-notices]] states; the notice takes its subject from the kind. The notices are numbered in the order of the cancellations.

Reservations that do not meet both conditions are not touched: one that is checked in, one that is `reserved` but has `start` at or before the request time, and one that is missed, completed or already cancelled. A session that is checked in goes on.

The escalation lists the cancelled reservations in `cancelledReservationIds` in the order they were cancelled, which is the order of the notices, and the new work order in `workOrderId`. The lockout, the cancellations and the notices are part of the escalation and are not audited on their own.
