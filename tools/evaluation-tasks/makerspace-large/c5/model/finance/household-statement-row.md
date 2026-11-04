---
kind: std/Concept
---
# Household statement row

One household member's row of a household statement.

**Fields**, in the formats of [[values]]:

- `memberId`: identifier, naming an existing [[members/member]]
- `invoiceId`: identifier, or `null` (the member's invoice for the month; `null` when the member has none)
- `totalCents`: cent amount (the invoice's `totalCents`, and 0 without an invoice)
- `balanceCents`: cent amount (the invoice's `balanceCents` at the request time, and 0 without an invoice)
