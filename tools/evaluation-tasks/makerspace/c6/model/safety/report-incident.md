---
kind: Operation
---
# Report incident

Creates an incident with the next incident identifier, status `open`, `reportedBy` the actor (a steward's identifier when a steward reports it), `reportedAt` the request time, and `resolution` and `resolvedAt` `null`.

- **Wire name:** `reportIncident`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `machineId`: identifier, naming an existing [[equipment/machine]]
  - `severity`: one of `low`, `medium` or `high`
  - `description`: text
- **Result:**
  - `incident`: a [[safety/incident]] record
- **Permissions:** staff all, admin all, steward all.
- **Audit target:** the new `incidentId`.
