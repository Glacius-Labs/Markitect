---
kind: Interface
---
# Ledger interface

What other modules may use of the ledger: its operations, the journal entry, account and trial balance row records, and the refusal `same_account`. Other modules also ask the ledger to post an entry and ask it for two figures of the journal.

- **Operations:** [[ledger/list-journal-entries]], [[ledger/get-trial-balance]], [[ledger/post-adjustment]].
- **Records:** [[ledger/journal-entry]], [[ledger/ledger-account]], [[ledger/trial-balance-row]].
- **Capabilities** that are not operations:
  - **Post an entry.** The caller names a debit account, a credit account, an amount in cents and a memo. The ledger records one [[ledger/journal-entry]] with the next entry identifier and `at` the request time. An amount of 0 posts nothing and consumes no number. The caller names the accounts, amount and memo that [[ledger/postings|the postings table]] gives for its event; the ledger does not check that the event happened.
  - **Entries of a month.** The number of journal entries whose `at` lies in a month.
  - **Balanced.** Whether the total of all debits equals the total of all credits. It always does.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `same_account` | The debit account and the credit account of an adjustment are the same. |
