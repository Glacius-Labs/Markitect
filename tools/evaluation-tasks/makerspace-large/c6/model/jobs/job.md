---
kind: std/Concept
---
# Job

The work done for an accepted quote by a member, the maker, generated identifier `job-N`. It is `started` until the maker completes it. `makerCreditCents` is `null` until then.

**Fields**, in the formats of [[values]]:

- `jobId`: identifier
- `quoteId`: identifier, naming an existing [[quotes/quote]]
- `customerId`: identifier, naming an existing [[quotes/customer]]; the quote's customer
- `makerId`: identifier, naming an existing [[members/member]]
- `status`: `started` or `completed`
- `totalCents`: cent amount; the quote's `totalCents`
- `makerCreditCents`: cent amount, or `null`
- `paidCents`: cent amount; the sum of the payments recorded for the job, 0 at the start

A record is identified by `jobId`.
