---
kind: Operation
---
# Cancel reservation

The reservation gets status `cancelled`. There is no fee and no credit.

- **Wire name:** `cancelReservation`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `reservationId`: identifier, naming an existing [[bookings/reservation]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the reported status is not `reserved`.
- **Result:**
  - `reservation`: a [[bookings/reservation]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `reservationId`.
