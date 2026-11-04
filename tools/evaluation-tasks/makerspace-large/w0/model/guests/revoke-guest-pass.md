---
kind: Operation
---
# Revoke guest pass

The guest pass gets status `revoked`. No charge is taken back. A pass belongs to the actor when its member is the actor.

- **Wire name:** `revokeGuestPass`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `guestPassId`: identifier, naming an existing [[guests/guest-pass]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the pass is already revoked or its `date` is before the date of the request time.
- **Result:**
  - `guestPass`: a [[guests/guest-pass]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `guestPassId`.
