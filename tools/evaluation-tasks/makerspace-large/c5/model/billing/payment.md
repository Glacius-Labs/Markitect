---
kind: std/Concept
---
# Payment

A payment of one invoice, generated identifier `pay-N`.

**Fields**, in the formats of [[values]]:

- `paymentId`: identifier
- `invoiceId`: identifier, naming an existing [[billing/invoice]]
- `memberId`: identifier, naming an existing [[members/member]]
- `amountCents`: integer from `1` to `100000000` cents (an amount paid or credited in one request)
- `appliedCents`: cent amount
- `creditedCents`: cent amount
- `at`: timestamp

A record is identified by `paymentId`.
