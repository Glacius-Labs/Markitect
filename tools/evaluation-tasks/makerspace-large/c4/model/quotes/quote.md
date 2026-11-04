---
kind: std/Concept
---
# Quote

An offer to a customer for machine time and materials, generated identifier `quo-N`. It moves from `draft` to `sent` to `accepted` or `rejected`. `subtotalCents` and `totalCents` are 0, and `sentAt` and `validUntil` are `null`, until the quote is sent.

**Fields**, in the formats of [[values]]:

- `quoteId`: identifier
- `customerId`: identifier, naming an existing [[quotes/customer]]
- `title`: text
- `status`: a [[quotes/quote-status]]: `draft`, `sent`, `accepted` or `rejected`
- `lines`: a list of [[quotes/quote-line]] records, in the order they were added
- `subtotalCents`: cent amount
- `totalCents`: cent amount
- `sentAt`: timestamp, or `null`
- `validUntil`: date, or `null`; it can lie after 2099 ([[computed-times]])

A record is identified by `quoteId`.
