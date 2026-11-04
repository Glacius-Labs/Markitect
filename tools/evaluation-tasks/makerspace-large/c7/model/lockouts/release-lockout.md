---
kind: Operation
---
# Release lockout

The lockout gets `releasedAt` the request time. Other active lockouts of the machine stay active.

- **Wire name:** `releaseLockout`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `lockoutId`: identifier, naming an existing [[lockouts/lockout]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the lockout is already released.
  2. `lockout_not_manual` when the lockout's `source` is not `manual`.
  3. `not_lockout_holder` when the actor is not the lockout's `appliedBy` and the actor's role is not admin.
- **Result:**
  - `lockout`: a [[lockouts/lockout]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `lockoutId`.
