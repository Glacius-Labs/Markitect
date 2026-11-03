# Notes: money cluster

Modules: ledger, deposits, passes, promotions, scholarships, refunds, dunning, lockers, finance, and the changes to billing and the revenue report.

## (a) New generated identifiers

| Record | Format |
| --- | --- |
| Journal entry | `jrn-1`, `jrn-2`, ... |
| Deposit | `dep-1`, ... |
| Pass | `pas-1`, ... |
| Award | `aws-1`, ... |
| Refund | `rfd-1`, ... |
| Dunning case | `dun-1`, ... |
| Rental | `rnt-1`, ... |

## (b) New lists, for the list-order page

| List | Order |
| --- | --- |
| `listJournalEntries` -> `entries` | creation order |
| `getTrialBalance` -> `rows` | the listing order of the ledger accounts (ascending) |
| `listDeposits` -> `deposits` | creation order |
| `listMemberPasses` -> `passes` | creation order |
| `listMemberPromotions` -> `redemptions` | creation order (the order the member redeemed) |
| `listAwards` -> `awards` | creation order |
| `listRefunds` -> `refunds` | creation order |
| `listDunningCases` -> `cases` | creation order |
| `runDunning` -> `actions` | invoices by ascending invoice number (the N of `inv-N` as a number), the steps of one invoice in the order notice, fee, suspend |
| `billLockers` -> `charges` | `lockerId` ascending |
| `chargesByKind` -> `rows` | the kind order membership, usage, material, course, deposit, pass, locker, guest, lateFee, event |
| `agedReceivables` -> `rows` | ascending invoice number (numeric) |
| `householdStatement` -> `rows` | `memberId` ascending |

Ledger entries of one request are numbered in the order of the rows of `ledger/postings`.

## (c) What I need from other modules' interfaces

- `notices-interface`: post a notice for a member with `kind`, `subject` and `ref` (dunning posts kinds `dunning` and `suspension`).
- `members-interface`: a way to set a member's standing from inside another operation (dunning), with no audit entry of its own.
- `households-interface`: the number of members of a member's household at a request time (billing, `household-discount`), a household's member ids ascending at the request time (finance), and the `households/household` record.
- Every module that posts uses the ledger capability "Post an entry"; every module that gives credit uses the billing capability "Add credit" and names the debit account of `ledger/postings`.
- `training` and `materials` use the capabilities of `promotions` and `scholarships` as `redeemed-promotions` and `course-discount` state them; `bookings` uses the draw of `passes`.
- Shared authorization: `rentalId` (used by `endLockerRental`, member own). `depositId` is not used with `own` by my operations.
- Shared pages: `initial-state` needs the seven prefixes above; `refusal-codes` needs nothing; `computed-times` (see d, item 3).

## (d) Gaps in the design and what I chose

1. `revenueReport`: "charges of new kinds are in none of its figures" cannot hold for `outstandingCents`, `creditAppliedCents` and `paymentsCents`, which come from invoice totals and payments. I wrote: the four kind sums and `invoicedCents` leave out other kinds; the other three take invoices and payments as they are. Old behavior is unchanged.
2. `closeStocktake` postings: shrink and gain are one pass over the counts in ascending `materialId`; each count posts one entry (shrink or gain), a difference of 0 posts none. `receiveOrder` posts the quantity received in that request times the line's unit cost.
3. `computed-times` says the latest `expiresOn` is `2100-12-31`; a pass with `validDays` 3650 sold on `2099-12-31` expires `2109-12-29`. The integrator should correct that sentence.
4. Dunning: a case is created in the first run that takes a step for the invoice, not when the invoice first becomes overdue. Notice subjects are fixed: `Invoice <invoiceId> is overdue` and `Suspended for overdue invoice <invoiceId>`. Days overdue use the timestamps (whole 1440-minute periods); `agedReceivables` uses calendar dates.
5. `chargesByKind`: `totalCents` is the sum of `invoicedCents` over the rows; no refund total is added. Result lists are named `rows` in every report (also the trial balance).
6. The ledger account page is `ledger/ledger-account` (not `ledger/account`) so that it cannot be mistaken for `billing/account`.
7. `trialBalanced` is always true (the invariant holds at all times); it is stated as such.
8. `lockers`: `already_billed` holds after any accepted `billLockers` of that month, even one that created no charge. `rental_limit` counts the member's active rentals of any locker.
9. Redemption order, award and fund details: a fund's `spentCents` stays after an award is revoked. An award with a discount of 0 cents still wins on percent, as the percent comparison is the rule.
10. `ledger` entry `amountCents` has no upper bound (a received order can exceed 100000000); `postAdjustment` takes 1 to 100000000.
