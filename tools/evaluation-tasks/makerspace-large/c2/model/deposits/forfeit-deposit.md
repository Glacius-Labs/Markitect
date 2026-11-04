---
kind: Operation
---
# Forfeit deposit

Forfeits the held deposit: its status becomes `forfeited` and `closedAt` the request time. The member's credit balance does not change. The `reason` is not stored. The entry is posted as [[ledger/postings|the postings of the ledger]] state.

A deposit is paid as [[deposits/release-deposit]] states.

Rules of other modules that apply here: [[ledger/postings]].

- **Wire name:** `forfeitDeposit`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `depositId`: identifier, naming an existing [[deposits/deposit]]
  - `reason`: text
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the deposit's status is not `held`.
  2. `deposit_unpaid` when the deposit is not paid.
- **Result:**
  - `deposit`: a [[deposits/deposit]] record
- **Permissions:** admin all.
- **Audit target:** `depositId`.
