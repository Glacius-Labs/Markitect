---
kind: Operation
---
# Check in

Status `checkedIn`; `checkedInAt` is the request time; `rateCentsPerMinute` is the machine's current rate. A missed, cancelled, checked-in or completed reservation is refused with `invalid_state`.

- **Wire name:** `checkIn`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `reservationId`: identifier, naming an existing [[bookings/reservation]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the reported status is not `reserved`.
  2. `check_in_too_early` when the request time is before `start`.
  3. `member_suspended` when the member is not in good standing ([[members/members-interface]]).
  4. `not_certified` when the member has no certification for the machine's family that is not revoked ([[training/training-interface]]).
- **Result:**
  - `reservation`: a [[bookings/reservation]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `reservationId`.
