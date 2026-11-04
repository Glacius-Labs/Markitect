---
kind: Operation
---
# Charge no-shows

Creates the no-show charges that are due at the request time, as [[bookings/no-show-fee|the no-show fee rule]] states. `charges` lists the charges created, in ascending reservation number. It is empty when nothing is due.

Rules of other modules that apply here: [[lockouts/lockouts-interface]] and [[billing/charges-and-invoices]].

- **Wire name:** `chargeNoShows`, a mutation.
- **Arguments:** only `actor` and `at`.
- **Result:**
  - `charges`: a list of [[billing/charge]] records
- **Permissions:** staff all, admin all.
- **Audit target:** the fixed string `noShows`.
