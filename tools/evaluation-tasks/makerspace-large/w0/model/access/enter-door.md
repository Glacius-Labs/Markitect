---
kind: Operation
---
# Enter door

The badge is inside. A door event is created with the next door event identifier: `kind` `enter`, `badgeId` the badge, `guestPassId` `null`, `at` the request time. The `badge` result is the badge after the change.

The request time is in the opening period of its date as [[bookings/opening-hours]] states: the opening time is in, the closing time is out, and on a Sunday no time is in. The member's overdue balance is the `overdueBalanceCents` of the member's account at the request time ([[billing/get-account]]).

Rules of other modules that apply here: [[members/membership]], [[bookings/opening-hours]] and [[billing/overdue-invoices]].

- **Wire name:** `enterDoor`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `badgeId`: identifier, naming an existing [[access/badge]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the badge is revoked.
  2. `already_inside` when the badge is inside.
  3. `member_suspended` when the badge's member is not in good standing ([[members/members-interface]]).
  4. `outside_opening_hours` when the request time is not in the opening period of its date.
  5. `account_overdue` when the member's overdue balance is above 0.
- **Result:**
  - `badge`: a [[access/badge]] record
  - `event`: a [[access/door-event]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `badgeId`.
