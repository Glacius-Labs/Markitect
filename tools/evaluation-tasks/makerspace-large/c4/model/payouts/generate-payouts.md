---
kind: Operation
---
# Generate payouts

Creates the payouts of the month and marks the month as generated, even when no payout results. For every user who has a rate, in ascending `userId` order, and has at least one scheduled class whose `start` lies in the month, a payout is created with the next payout identifier, status `open`, and `minutes` and `amountCents` as [[payouts/payout-calculation]] states. A user without a rate, and a user with no such class, gets no payout. A payout of 0 cents is still created. For each payout a posting is made as [[ledger/postings]] states.

Rules of other modules that apply here: [[ledger/postings]].

- **Wire name:** `generatePayouts`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `month`: month
- **Refusals** at check 7, in this order:
  1. `month_not_ended` when the request time is before the first minute of the following month (for `2026-03`, before `2026-04-01T00:00Z`). The month `2099-12` never ends at a request time.
  2. `already_generated` when payouts for this month were generated before.
- **Result:**
  - `month`: month
  - `payouts`: a list of [[payouts/payout]] records
- **Permissions:** admin all.
- **Audit target:** `month`.
