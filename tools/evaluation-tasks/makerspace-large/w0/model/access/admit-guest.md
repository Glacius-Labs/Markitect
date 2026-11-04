---
kind: Operation
---
# Admit guest

Admits the guest of the pass. A door event is created with the next door event identifier: `kind` `guest`, `badgeId` the badge, `guestPassId` the pass, `at` the request time. The badge does not change. A pass admits its guest once. Ownership is judged on `badgeId` alone: the badge's member is the actor. Standing, opening hours and the account are not checked.

- **Wire name:** `admitGuest`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `guestPassId`: identifier, naming an existing [[guests/guest-pass]]
  - `badgeId`: identifier, naming an existing [[access/badge]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the guest pass is revoked or the badge is revoked.
  2. `pass_not_today` when the pass's `date` is not the date of the request time.
  3. `pass_of_another_member` when the pass's member is not the badge's member.
  4. `sponsor_not_inside` when the badge is not inside.
  5. `guest_already_admitted` when a door event of kind `guest` exists for this pass.
- **Result:**
  - `event`: a [[access/door-event]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `guestPassId`.
