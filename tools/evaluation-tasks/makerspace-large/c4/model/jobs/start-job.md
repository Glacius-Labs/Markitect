---
kind: Operation
---
# Start job

Creates a [[jobs/job]] for the accepted quote with the next job identifier, status `started`, `customerId` and `totalCents` those of the quote, `makerCreditCents` `null` and `paidCents` 0.

Rules of other modules that apply here: [[jobs/maker-gate]], [[members/membership]] and [[training/certifications]].

- **Wire name:** `startJob`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `quoteId`: identifier, naming an existing [[quotes/quote]]
  - `makerId`: identifier, naming an existing [[members/member]]
- **Refusals** at check 7, in this order:
  1. `quote_not_accepted` when the quote's status is not `accepted`.
  2. `already_started` when a job exists for the quote.
  3. `member_suspended` when the maker is not in good standing ([[members/members-interface]]).
  4. `maker_not_certified` when the quote has a `machine` line and the maker has no certification valid on the date of the request time for the family of the machine of some such line ([[jobs/maker-gate]]).
- **Result:**
  - `job`: a [[jobs/job]] record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `jobId`.
