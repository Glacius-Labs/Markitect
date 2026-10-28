---
kind: std/Concept
---
# Invoice

An invoice of one member for one month, generated identifier `inv-N`.

**Fields**, in the formats of [[values]]:

- `invoiceId`: identifier
- `memberId`: identifier, naming an existing [[members/member]]
- `month`: month
- `issuedAt`: timestamp
- `dueAt`: timestamp
- `lines`: a list of [[billing/charge]] records
- `totalCents`: cent amount
- `creditAppliedCents`: cent amount
- `paidCents`: cent amount
- `balanceCents`: cent amount
- `status`: `open` or `paid`
- `overdue`: flag

A record is identified by `invoiceId`.
