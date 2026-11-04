---
kind: Operation
---
# Send quote

Prices and sends the quote: its status becomes `sent`, every line gets its `unitPriceCents`, and `subtotalCents`, `totalCents`, `sentAt` and `validUntil` are set, all as [[quotes/quote-pricing]] and [[quotes/quote-validity]] state.

Rules of other modules that apply here: [[quotes/quote-pricing]], [[quotes/quote-validity]] and [[money]].

- **Wire name:** `sendQuote`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `quoteId`: identifier, naming an existing [[quotes/quote]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the quote's status is not `draft`.
  2. `empty_quote` when the quote has no line.
- **Result:**
  - `quote`: a [[quotes/quote]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `quoteId`.
