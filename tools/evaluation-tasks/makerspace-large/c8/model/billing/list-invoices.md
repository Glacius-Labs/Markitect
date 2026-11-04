---
kind: Operation
---
# List invoices

Returns all of the member's invoices.

- **Wire name:** `listInvoices`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
- **Result:**
  - `memberId`: identifier
  - `invoices`: a list of [[billing/invoice]] records
- **Permissions:** member own, staff all, admin all.
