---
kind: Operation
---
# Create reservation

Creates the reservation with status `reserved` and the next reservation identifier.

Examples: on Sunday `2026-03-08`, `12:00`–`12:15` is accepted and `11:45`–`12:15` is refused with `outside_opening_hours`. On Friday `2026-03-06`, `19:45`–`20:00` is inside opening hours and `19:45`–`20:15` is not. On Tuesday `2026-03-03`, `10:15`–`10:30` is accepted and `10:10`–`10:30` is refused with `invalid_slot`.

- **Wire name:** `createReservation`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
  - `machineId`: identifier, naming an existing [[equipment/machine]]
  - `start`: timestamp
  - `end`: timestamp
- **Refusals** at check 7, in this order:
  1. `invalid_interval` when `end` is not after `start`.
  2. `invalid_slot` when `start` or `end` is not on a 15-minute boundary or the length is outside 15 to 240 minutes.
  3. `starts_in_past` when `start` is before the request time.
  4. `outside_opening_hours` when the slot is not inside the opening hours of its start date.
  5. `member_suspended` when the member is not in good standing ([[members/members-interface]]).
  6. `account_overdue` when the member is blocked ([[billing/billing-interface]]).
  7. `not_certified` when the member has no certification for the machine's family that is not revoked ([[training/training-interface]]).
  8. `machine_in_maintenance` when a scheduled maintenance window of the machine overlaps the slot.
  9. `machine_locked` when a lockout of the machine overlaps the slot.
  10. `machine_booked` when another active reservation of the machine overlaps the slot.
  11. `member_double_booked` when another active reservation of the member overlaps the slot.
- **Result:**
  - `reservation`: a [[bookings/reservation]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** the new `reservationId`.
