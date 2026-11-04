---
kind: Operation
---
# Create reservation

Creates the reservation with status `reserved` and the next reservation identifier.

Example: on Saturday `2026-03-07`, the slot `17:30`–`18:00` is inside opening hours and `17:30`–`18:30` is not; on Tuesday `2026-03-03`, `10:10`–`11:00` is refused with `invalid_slot`.

Example of the order: a machine that is locked out and also due for inspection refuses a reservation with `machine_locked_out`, because that refusal comes first.

Rules of other modules that apply here: [[guardians/guardian-consent]], [[waivers/waiver-gate]], [[lockouts/lockout-gate]], [[inspections/inspection-gate]] and [[staffing/supervision]].

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
  6. `guardian_consent_missing` when the member is a minor on the date of `start` and holds no consent for the machine's family ([[guardians/guardian-consent]]).
  7. `not_certified` when the member has no certification for the machine's family valid on the date of `start` ([[training/training-interface]]).
  8. `waiver_missing` when the [[waivers/waiver-gate]] refuses the member for the machine's family.
  9. `machine_locked_out` when the machine has an active lockout at the request time, whatever the date of the slot ([[lockouts/lockout-gate]]).
  10. `inspection_due` when the machine has an inspection plan and the date of `start` is on or after its `nextDueOn` ([[inspections/inspection-gate]]).
  11. `machine_in_maintenance` when a scheduled maintenance window of the machine overlaps the slot.
  12. `supervisor_required` when the machine's family is supervised and the slot is not wholly inside the union of the scheduled shifts ([[staffing/supervision]]).
  13. `machine_booked` when another active reservation of the machine overlaps the slot.
  14. `member_double_booked` when another active reservation of the member overlaps the slot.
- **Result:**
  - `reservation`: a [[bookings/reservation]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** the new `reservationId`.
