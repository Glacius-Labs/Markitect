---
kind: Operation
---
# List refunds

Returns all refunds of the invoice, in creation order (ascending refund number). An invoice without refunds gives an empty list.

- **Wire name:** `listRefunds`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `invoiceId`: identifier, naming an existing [[billing/invoice]]
- **Result:**
  - `invoiceId`: identifier
  - `refunds`: a list of [[refunds/refund]] records
- **Permissions:** member own, staff all, admin all.
