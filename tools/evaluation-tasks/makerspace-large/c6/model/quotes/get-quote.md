---
kind: Operation
---
# Get quote

Returns the quote with its lines in the order they were added.

- **Wire name:** `getQuote`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `quoteId`: identifier, naming an existing [[quotes/quote]]
- **Result:**
  - `quote`: a [[quotes/quote]] record
- **Permissions:** staff all, admin all.
