---
kind: Operation
---
# Create quote

Creates a [[quotes/quote]] for the customer with the next quote identifier, status `draft`, no lines, `subtotalCents` and `totalCents` 0, and `sentAt` and `validUntil` `null`.

- **Wire name:** `createQuote`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `customerId`: identifier, naming an existing [[quotes/customer]]
  - `title`: text
- **Result:**
  - `quote`: a [[quotes/quote]] record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `quoteId`.
