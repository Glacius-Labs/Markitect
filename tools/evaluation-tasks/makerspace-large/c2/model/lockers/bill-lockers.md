---
kind: Operation
---
# Bill lockers

Charges the rent for the month. One locker [[billing/charge]] is created for every rental with status `active` at the request time, in ascending `lockerId` order: `kind` `locker`, `ref` the rental identifier, `quantity` 1, `unitPriceCents` the locker's `monthlyCents` at the request time, `discountPercent` 0 and `at` the request time. The whole rent is charged, whenever in the month the rental started and whatever the member's standing is. The month is then marked as billed, even when there was no active rental. `charges` lists the charges created; it is empty when there was no active rental.

Rules of other modules that apply here: [[billing/charges-and-invoices]].

- **Wire name:** `billLockers`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `month`: month
- **Refusals** at check 7, in this order:
  1. `not_current_month` when `month` is not the month of the request time.
  2. `already_billed` when an earlier request to bill the lockers of this month was accepted.
- **Result:**
  - `month`: month
  - `charges`: a list of [[billing/charge]] records
- **Permissions:** staff all, admin all.
- **Audit target:** `month`.
