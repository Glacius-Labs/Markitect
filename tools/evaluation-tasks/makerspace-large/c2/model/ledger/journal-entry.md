---
kind: std/Concept
---
# Journal entry

One posting of the ledger: an amount that debits one [[ledger/ledger-account|account]] and credits another, generated identifier `jrn-N`. Entries are never changed or removed. Which events post which entries is stated in [[ledger/postings]].

**Fields**, in the formats of [[values]]:

- `entryId`: identifier
- `at`: timestamp (the request time of the posting)
- `debitAccount`: a [[ledger/ledger-account]]
- `creditAccount`: a [[ledger/ledger-account]]
- `amountCents`: integer of at least `1` cents (a posted amount; an amount of 0 is never posted)
- `memo`: text (what the posting is for: the identifier the postings table names)

A record is identified by `entryId`.
