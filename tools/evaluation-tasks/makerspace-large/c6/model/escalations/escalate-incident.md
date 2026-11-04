---
kind: Operation
---
# Escalate incident

Creates an escalation with the next escalation identifier, `incidentId` as supplied, `machineId` the incident's machine, `workOrderId` the work order it opens, `cancelledReservationIds` the reservations it cancels and `at` the request time. It has the effects that [[escalations/escalation-effects]] states. The incident itself is not changed: it stays `open` until it is resolved.

- **Wire name:** `escalateIncident`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `incidentId`: identifier, naming an existing [[safety/incident]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the incident's `status` is `resolved`.
  2. `already_escalated` when an escalation of the incident exists.
  3. `no_policy` when the family of the incident's machine has no [[escalations/escalation-policy]].
  4. `below_policy` when the incident's `severity` is lower than the policy's `minSeverity`, with `low` < `medium` < `high`. An equal severity is not below.
- **Result:**
  - `escalation`: an [[escalations/escalation]] record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `escalationId`.
