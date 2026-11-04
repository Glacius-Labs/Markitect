---
kind: Operation
---
# Add fund

Creates the fund with `spentCents` 0.

- **Wire name:** `addFund`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `fundId`: identifier
  - `budgetCents`: integer from `1` to `100000000` cents (the budget of a fund)
- **Refusals** at check 7, in this order:
  1. `already_exists` when a fund with `fundId` exists.
- **Result:**
  - `fund`: a [[scholarships/fund]] record
- **Permissions:** admin all.
- **Audit target:** `fundId`.
