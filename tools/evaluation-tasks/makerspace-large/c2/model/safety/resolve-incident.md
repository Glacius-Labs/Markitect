---
kind: Operation
---
# Resolve incident

Status `resolved`, `resolution` as supplied, `resolvedAt` the request time.

- **Wire name:** `resolveIncident`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `incidentId`: identifier, naming an existing [[safety/incident]]
  - `resolution`: text
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the incident is already resolved.
- **Result:**
  - `incident`: a [[safety/incident]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `incidentId`.
