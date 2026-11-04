---
kind: Operation
---
# Rent locker

Rents the locker to the member. A rental is created with the next rental identifier, status `active`, `startedAt` the request time and `endedAt` `null`. Nothing is charged by this request; [[lockers/bill-lockers]] charges the rent.

Rules of other modules that apply here: [[members/membership]].

- **Wire name:** `rentLocker`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
  - `lockerId`: identifier, naming an existing [[lockers/locker]]
- **Refusals** at check 7, in this order:
  1. `member_suspended` when the member is not in good standing ([[members/members-interface]]).
  2. `locker_taken` when the locker has a rental with status `active`.
  3. `rental_limit` when the member has 2 rentals with status `active`.
- **Result:**
  - `rental`: a [[lockers/rental]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** the new `rentalId`.
