---
kind: Operation
---
# Leave door

The badge is no longer inside. A door event is created with the next door event identifier: `kind` `leave`, `badgeId` the badge, `guestPassId` `null`, `at` the request time. Standing, opening hours and the account are not checked. The `badge` result is the badge after the change.

- **Wire name:** `leaveDoor`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `badgeId`: identifier, naming an existing [[access/badge]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the badge is revoked.
  2. `not_inside` when the badge is not inside.
- **Result:**
  - `badge`: a [[access/badge]] record
  - `event`: a [[access/door-event]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `badgeId`.
