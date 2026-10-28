---
kind: Operation
---
# Generate invoices

Creates the invoices as [[billing/issuing-invoices|issuing invoices]] describes and marks the month as invoiced, even when there is no member to invoice.

Rules of other modules that apply here: [[members/membership]].

- **Wire name:** `generateInvoices`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `month`: month
- **Refusals** at check 7, in this order:
  1. `month_not_ended` when the request time is before the first minute of the following month (for `2026-03`, before `2026-04-01T00:00Z`).
  2. `already_invoiced` when invoices for this month were generated before.
- **Result:**
  - `month`: month
  - `invoices`: a list of [[billing/invoice]] records
- **Permissions:** staff all, admin all.
- **Audit target:** `month`.
