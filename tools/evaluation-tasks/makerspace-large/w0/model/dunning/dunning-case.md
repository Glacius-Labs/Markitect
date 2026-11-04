---
kind: std/Concept
---
# Dunning case

The record of the steps taken for one overdue invoice, generated identifier `dun-N`. There is at most one case per invoice. It is created in the first run that takes a step for the invoice ([[dunning/dunning-run]]). Each timestamp is `null` until its step is taken, and is then the request time of the run that took it.

**Fields**, in the formats of [[values]]:

- `caseId`: identifier
- `invoiceId`: identifier, naming an existing [[billing/invoice]]
- `memberId`: identifier, naming an existing [[members/member]] (the invoice's member)
- `noticedAt`: timestamp, or `null`
- `feeChargedAt`: timestamp, or `null`
- `suspendedAt`: timestamp, or `null`

A record is identified by `caseId`.
