---
kind: std/Concept
---
# Refund

A refund of part or all of one line of a paid invoice, given as credit, generated identifier `rfd-N`. The invoice and its line are not changed.

**Fields**, in the formats of [[values]]:

- `refundId`: identifier
- `invoiceId`: identifier, naming an existing [[billing/invoice]]
- `lineNumber`: integer from `1` to `1000` (the position of the line in the invoice's `lines`; 1 is the first line)
- `amountCents`: integer from `1` to `100000000` cents (a refund amount)
- `reason`: text
- `at`: timestamp

A record is identified by `refundId`.
