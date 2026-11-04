---
kind: Operation
---
# Issue credit

The member's credit balance increases by `amountCents`. The credit counts as issued at the request time for the revenue report. `creditBalanceCents` is the member's credit balance after the change. The entry of the ledger is posted as [[ledger/postings|the postings of the ledger]] state.

Rules of other modules that apply here: [[ledger/postings]].

- **Wire name:** `issueCredit`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
  - `amountCents`: integer from `1` to `100000000` cents (an amount paid or credited in one request)
  - `reason`: text
- **Result:**
  - `memberId`: identifier
  - `amountCents`: integer from `1` to `100000000` cents (an amount paid or credited in one request)
  - `creditBalanceCents`: cent amount
- **Permissions:** admin all.
- **Audit target:** `memberId`.
