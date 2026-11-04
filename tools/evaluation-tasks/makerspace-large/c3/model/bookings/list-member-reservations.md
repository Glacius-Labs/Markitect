---
kind: Operation
---
# List member reservations

Returns all of the member's reservations in every status.

- **Wire name:** `listMemberReservations`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
- **Result:**
  - `memberId`: identifier
  - `reservations`: a list of [[bookings/reservation]] records
- **Permissions:** member own, staff all, admin all.
