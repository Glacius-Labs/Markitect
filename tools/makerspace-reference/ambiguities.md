# Readings chosen where the neutral text is open

Each line: sentence (section), reading implemented. Reviewer findings are not yet applied.

1. 2.5 / 3.1 check 2: an `actor` that is not a valid identifier is refused `invalid_argument`, not `unknown_actor`.
2. 2.4: an empty or whitespace-only line gets an `invalidRequest` response with `id` null.
3. 11.1: `generateInvoices` for `2099-12` can never succeed, because the month ends after the last valid timestamp (`month_not_ended`).
4. 7.2 / 11.1: dates computed past 2099 (an `expiresOn` or `dueAt` after 2099-12-31) are printed as such.
5. 11.1: unbilled charges of a month that was not invoiced stay unbilled when a later month is invoiced; only charges whose `at` lies in the invoiced month go on its invoice.
6. 9.1: a checked-in reservation stays active (blocks its interval) after its end and is never reported `missed`.
7. 13: "range longer than 366 days" means more than 366 days between `from` and `to`.
8. 13: `paymentsCents` counts payments by the month of their `at`, whatever the invoice's month.
9. 3.1 check 5: every `memberId` argument that refers to an existing record must name a user with role `member` (also in `revokeCertification`, `listCertifications`), otherwise `not_found`.
10. C2: the cancelled set uses the reported status at the request time (a reservation whose end has passed is `missed` and is not cancelled).
11. C1: a `forbidden` refusal is recorded only after checks 1 to 3 passed; `time_regression` and earlier are not recorded.
