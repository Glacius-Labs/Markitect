---
kind: Operation
---
# End locker rental

Ends the rental: its status becomes `ended` and `endedAt` the request time. The locker can be rented again. Charges already created stay. The member's standing is not checked.

- **Wire name:** `endLockerRental`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `rentalId`: identifier, naming an existing [[lockers/rental]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the rental's status is `ended`.
- **Result:**
  - `rental`: a [[lockers/rental]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `rentalId`.
