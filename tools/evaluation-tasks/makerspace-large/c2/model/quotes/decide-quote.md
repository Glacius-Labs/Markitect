---
kind: Operation
---
# Decide quote

The quote's status becomes `accepted` or `rejected`, as `decision` says. Nothing else of the quote changes.

Rules of other modules that apply here: [[quotes/quote-validity]].

- **Wire name:** `decideQuote`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `quoteId`: identifier, naming an existing [[quotes/quote]]
  - `decision`: `accepted` or `rejected`
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the quote's status is not `sent`.
  2. `quote_expired` when `decision` is `accepted` and the date of the request time is after `validUntil`.
- **Result:**
  - `quote`: a [[quotes/quote]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `quoteId`.
