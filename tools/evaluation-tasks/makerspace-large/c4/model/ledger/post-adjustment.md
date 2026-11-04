---
kind: Operation
---
# Post adjustment

Posts one journal entry as given: the next entry identifier, `at` the request time, and the debit account, credit account, amount and memo of the arguments. Nothing else changes. After an adjustment names an account on either side, the invariants of [[ledger/postings]] that concern that account are no longer promised.

- **Wire name:** `postAdjustment`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `debitAccount`: an account of [[ledger/ledger-account]], one of `cash`, `creditLiability`, `depositLiability`, `expenseGoodwill`, `expenseMaker`, `expensePayout`, `expenseVolunteer`, `expenseWriteOff`, `inventory`, `payable`, `receivable`, `revenueCourse`, `revenueDeposits`, `revenueEvents`, `revenueGuests`, `revenueJobs`, `revenueLateFees`, `revenueLockers`, `revenueMaterial`, `revenueMembership`, `revenueNoShows`, `revenuePasses`, `revenueRefunds` or `revenueUsage`
  - `creditAccount`: an account of [[ledger/ledger-account]], one of `cash`, `creditLiability`, `depositLiability`, `expenseGoodwill`, `expenseMaker`, `expensePayout`, `expenseVolunteer`, `expenseWriteOff`, `inventory`, `payable`, `receivable`, `revenueCourse`, `revenueDeposits`, `revenueEvents`, `revenueGuests`, `revenueJobs`, `revenueLateFees`, `revenueLockers`, `revenueMaterial`, `revenueMembership`, `revenueNoShows`, `revenuePasses`, `revenueRefunds` or `revenueUsage`
  - `amountCents`: integer from `1` to `100000000` cents (an amount posted in one adjustment)
  - `memo`: text
- **Refusals** at check 7, in this order:
  1. `same_account` when `debitAccount` equals `creditAccount`.
- **Result:**
  - `entry`: a [[ledger/journal-entry]] record
- **Permissions:** admin all.
- **Audit target:** the new `entryId`.
