---
kind: Operation
---
# Create reservation

Creates the reservation with status `reserved` and the next reservation identifier.

Example: on Saturday `2026-03-07`, the slot `17:30`–`18:00` is inside opening hours and `17:30`–`18:30` is not; on Tuesday `2026-03-03`, `10:15`–`11:00` is refused with `invalid_slot`.

- **Wire name:** `createReservation`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
  - `machineId`: identifier, naming an existing [[equipment/machine]]
  - `start`: timestamp
  - `end`: timestamp
- **Refusals** at check 7, in this order:
  1. `invalid_interval` when `end` is not after `start`.
  2. `invalid_slot` when `start` or `end` is not on a 30-minute boundary or the length is outside 30 to 240 minutes.
  3. `starts_in_past` when `start` is before the request time.
  4. `outside_opening_hours` when the slot is not inside the opening hours of its start date.
  5. `member_suspended` when the member is not in good standing ([[members/members-interface]]).
  6. `not_certified` when the member has no certification for the machine's family valid on the date of `start` ([[training/training-interface]]).
  7. `machine_in_maintenance` when a scheduled maintenance window of the machine overlaps the slot.
  8. `machine_booked` when another active reservation of the machine overlaps the slot.
  9. `member_double_booked` when another active reservation of the member overlaps the slot.
- **Result:**
  - `reservation`: a [[bookings/reservation]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** the new `reservationId`.
