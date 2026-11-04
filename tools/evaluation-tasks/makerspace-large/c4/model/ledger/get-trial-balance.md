---
kind: Operation
---
# Get trial balance

Returns one row for every [[ledger/ledger-account|account]], including accounts without entries, calculated from all journal entries at the request time. The totals are the sums over all rows. Because every entry debits one account and credits another by the same amount, the two totals are always equal ([[ledger/postings]]).

- **Wire name:** `getTrialBalance`, a read.
- **Arguments:** only `actor` and `at`.
- **Result:**
  - `rows`: a list of [[ledger/trial-balance-row]] records, one per account, in the listing order of [[ledger/ledger-account]]
  - `totalDebitCents`: cent amount (the sum of `debitCents` over the rows)
  - `totalCreditCents`: cent amount (the sum of `creditCents` over the rows)
- **Permissions:** admin all.
