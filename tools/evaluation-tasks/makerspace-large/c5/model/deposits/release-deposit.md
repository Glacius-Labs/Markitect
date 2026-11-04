---
kind: Operation
---
# Release deposit

Releases the held deposit: its status becomes `released` and `closedAt` the request time. The member's credit balance increases by the deposit's `amountCents`, and the entry is posted as [[ledger/postings|the postings of the ledger]] state. `creditBalanceCents` is the member's credit balance after the change. The member's standing is not checked.

A deposit is paid when the invoice that has its charge (the line with `kind` `deposit` and `ref` the deposit identifier) has status `paid`. A deposit whose charge is not yet on any invoice is not paid.

Rules of other modules that apply here: [[ledger/postings]] and [[billing/charges-and-invoices]].

- **Wire name:** `releaseDeposit`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `depositId`: identifier, naming an existing [[deposits/deposit]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the deposit's status is not `held`.
  2. `deposit_unpaid` when the deposit is not paid.
- **Result:**
  - `deposit`: a [[deposits/deposit]] record
  - `creditBalanceCents`: cent amount
- **Permissions:** staff all, admin all.
- **Audit target:** `depositId`.
