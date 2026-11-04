---
kind: Operation
---
# Apply lockout

Creates a lockout of the machine with the next lockout identifier, `source` `manual`, `reason` as supplied, `appliedBy` the actor, `appliedAt` the request time and `releasedAt` `null`. The machine may already have active lockouts; they stay.

Rules of other modules that apply here: [[lockouts/lockout-gate]].

- **Wire name:** `applyLockout`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `machineId`: identifier, naming an existing [[equipment/machine]]
  - `reason`: text
- **Result:**
  - `lockout`: a [[lockouts/lockout]] record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `lockoutId`.
