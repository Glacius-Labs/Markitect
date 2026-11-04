---
kind: Operation
---
# Check in

Status `checkedIn`; `checkedInAt` is the request time; `rateCentsPerMinute` is the machine's current rate. A missed, cancelled, checked-in or completed reservation is refused with `invalid_state`.

Rules of other modules that apply here: [[guardians/guardian-consent]], [[waivers/waiver-gate]], [[ppe/ppe-gate]], [[lockouts/lockout-gate]] and premises gate.

- **Wire name:** `checkIn`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `reservationId`: identifier, naming an existing [[bookings/reservation]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the reported status is not `reserved`.
  2. `check_in_too_early` when the request time is before `start`.
  3. `member_suspended` when the member is not in good standing ([[members/members-interface]]).
  4. `guardian_consent_missing` when the member is a minor on the date of the request time and holds no consent for the machine's family ([[guardians/guardian-consent]]).
  5. `not_certified` when the member has no certification for the machine's family valid on the date of the request time ([[training/training-interface]]).
  6. `waiver_missing` when the [[waivers/waiver-gate]] refuses the member for the machine's family.
  7. `ppe_missing` when the member does not hold every item that the machine's family requires ([[ppe/ppe-gate]]).
  8. `machine_locked_out` when the machine has an active lockout at the request time ([[lockouts/lockout-gate]]).
  9. `not_on_premises` when the member has an active badge that is not inside (premises gate).
- **Result:**
  - `reservation`: a [[bookings/reservation]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `reservationId`.
