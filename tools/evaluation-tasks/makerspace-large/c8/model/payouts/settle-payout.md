---
kind: Operation
---
# Settle payout

The payout gets status `settled`. A posting is made as [[ledger/postings]] states. A payout of 0 cents can be settled; it posts nothing.

Rules of other modules that apply here: [[ledger/postings]].

- **Wire name:** `settlePayout`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `payoutId`: identifier, naming an existing [[payouts/payout]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the payout is already settled.
- **Result:**
  - `payout`: a [[payouts/payout]] record
- **Permissions:** admin all.
- **Audit target:** `payoutId`.
