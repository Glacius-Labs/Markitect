---
kind: std/Concept
---
# Loan

The loan of one tool to one member, generated identifier `lon-N`. A tool has at most one active loan. `returnedAt` is `null` while the loan is active.

**Fields**, in the formats of [[values]]:

- `loanId`: identifier
- `toolId`: identifier, naming an existing [[loans/tool]]
- `memberId`: identifier, naming an existing [[members/member]]
- `borrowedAt`: timestamp
- `dueOn`: date, 14 days after the date of `borrowedAt`
- `returnedAt`: timestamp, or `null`
- `status`: `active` or `returned`
- `overdue`: flag, reported at the request time of the response ([[loans/overdue-loans]])
- `chargedDays`: count, the days of late fee charged so far, 0 at the start ([[loans/late-fee-charge]])

A record is identified by `loanId`.
