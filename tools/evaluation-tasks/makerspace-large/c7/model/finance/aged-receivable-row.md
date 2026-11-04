---
kind: std/Concept
---
# Aged receivable row

One open invoice's row of the aged receivables report.

**Fields**, in the formats of [[values]]:

- `invoiceId`: identifier, naming an existing [[billing/invoice]]
- `memberId`: identifier, naming an existing [[members/member]]
- `balanceCents`: cent amount (the invoice's balance at the request time)
- `daysOverdue`: count (the whole days from the date of the invoice's `dueAt` to the report date, and 0 when the report date is not after that date)
