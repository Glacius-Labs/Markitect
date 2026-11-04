---
kind: std/Concept
---
# Trial balance row

One account's row of the trial balance.

**Fields**, in the formats of [[values]]:

- `account`: a [[ledger/ledger-account]]
- `debitCents`: cent amount (the sum of `amountCents` of all [[ledger/journal-entry|entries]] whose `debitAccount` is the account)
- `creditCents`: cent amount (the sum of `amountCents` of all entries whose `creditAccount` is the account)
- `balanceCents`: cent amount (the larger of `debitCents` and `creditCents` minus the smaller)
- `side`: `debit` when `debitCents` is larger, `credit` when `creditCents` is larger, `none` when they are equal
