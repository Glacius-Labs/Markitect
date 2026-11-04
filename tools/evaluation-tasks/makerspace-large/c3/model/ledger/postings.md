---
kind: std/Rule
---
# Postings

Every event in the table below posts [[ledger/journal-entry|journal entries]] as part of the operation that causes it. The entries are created together with the operation's other effects. An entry has `at` the request time, takes the next entry identifier, debits the **Debit** account and credits the **Credit** account by the **Amount**, and has the **Memo** as its `memo`. A refused request posts nothing.

An amount of 0 posts nothing and consumes no entry number. Entries are numbered in the order they are posted. Generate invoices takes the invoices in ascending `memberId` order and posts, for each invoice, its lines in order and then its credit entry, before the next invoice. Close stocktake takes the counts in ascending `materialId` order and posts each count's one entry. Every other operation posts in the order of its rows.

| Event | Debit | Credit | Amount | Memo |
| --- | --- | --- | --- | --- |
| `generateInvoices`, for each line of each invoice: invoices in ascending `memberId` order, the lines of an invoice in the order of its `lines` | `receivable` | the revenue account of the line's `kind`, from the table below | the line's `amountCents` | the `invoiceId` |
| `generateInvoices`, credit applied, once per invoice, after that invoice's lines | `creditLiability` | `receivable` | the invoice's `creditAppliedCents` | the `invoiceId` |
| `recordPayment` | `cash` | `receivable` | `appliedCents` | the `paymentId` |
| `recordPayment`, then | `cash` | `creditLiability` | `creditedCents` | the `paymentId` |
| `issueCredit` | `expenseGoodwill` | `creditLiability` | `amountCents` | the `memberId` |
| `releaseDeposit` | `depositLiability` | `creditLiability` | the deposit's `amountCents` | the `depositId` |
| `forfeitDeposit` | `depositLiability` | `revenueDeposits` | the deposit's `amountCents` | the `depositId` |
| `issueRefund` | `revenueRefunds` | `creditLiability` | `amountCents` | the `refundId` |
| `decideVolunteerTime`, approved | `expenseVolunteer` | `creditLiability` | the entry's `creditCents` | the `entryId` |
| `cancelTicket`, and `cancelEvent` for each ticket it cancels, in creation order | `revenueRefunds` | `creditLiability` | the credit, the event's `priceCents` | the `ticketId` |
| `receiveOrder` | `inventory` | `payable` | the quantity received in the request × the line's `unitCostCents` | the `orderId` |
| `closeStocktake`, for each count with a negative `difference`, the counts taken in one pass in ascending `materialId` order | `expenseWriteOff` | `inventory` | the units short × the material's `unitPriceCents` at the request time | the `stocktakeId` |
| `closeStocktake`, for each count with a positive `difference`, in the same pass | `inventory` | `expenseWriteOff` | the units over × the material's `unitPriceCents` at the request time | the `stocktakeId` |
| `generatePayouts`, for each payout, in creation order | `expensePayout` | `payable` | the payout's `amountCents` | the `payoutId` |
| `settlePayout` | `payable` | `cash` | the payout's `amountCents` | the `payoutId` |
| `completeJob` | `expenseMaker` | `creditLiability` | the job's `makerCreditCents` | the `jobId` |
| `recordJobPayment` | `cash` | `revenueJobs` | `amountCents` | the `jobId` |
| [[ledger/post-adjustment]] | as given | as given | as given | as given |

The revenue account of an invoice line, by the `kind` of the line:

| Line `kind` | Credit account |
| --- | --- |
| `membership` | `revenueMembership` |
| `usage` | `revenueUsage` |
| `material` | `revenueMaterial` |
| `course` | `revenueCourse` |
| `deposit` | `depositLiability` |
| `pass` | `revenuePasses` |
| `locker` | `revenueLockers` |
| `guest` | `revenueGuests` |
| `lateFee` | `revenueLateFees` |
| `event` | `revenueEvents` |
| `noShow` | `revenueNoShows` |

The credit of an operation that adds to a member's credit balance is posted as the row of that operation says, with `creditLiability` as the credit account. The units short at a stocktake are the absolute value of a negative `difference`.

**Invariants** the program holds after every request:

- The sum of all debits equals the sum of all credits, because every entry adds its amount to one debit and one credit.
- The `receivable` account's `debitCents` less its `creditCents` equals the sum of `balanceCents` over all invoices (an invoice that is `paid` has balance 0).
- The `creditLiability` account's `creditCents` less its `debitCents` equals the sum of all members' credit balances.

The second and third hold unless a [[ledger/post-adjustment|postAdjustment]] named that account on either side; from then on the difference between the two figures is the net of the adjustments.

**Worked example: an invoice with credit applied.** Member `ada` on tier `plus` is the only member. On `2026-03-10T09:00Z` an admin issues her 2000 cents of credit. Entry `jrn-1` debits `expenseGoodwill` and credits `creditLiability` by 2000, memo `ada`. She has a usage charge of 1499 cents in March (overdue invoices). Generating `2026-03` at `2026-04-01T08:00Z` creates `inv-1` with the membership line 4900 and the usage line 1499, `totalCents` 6399, `creditAppliedCents` 2000 and `balanceCents` 4399. It posts, in this order:

| Entry | Debit | Credit | Amount | Memo |
| --- | --- | --- | --- | --- |
| `jrn-2` | `receivable` | `revenueMembership` | 4900 | `inv-1` |
| `jrn-3` | `receivable` | `revenueUsage` | 1499 | `inv-1` |
| `jrn-4` | `creditLiability` | `receivable` | 2000 | `inv-1` |

`receivable` now has `debitCents` 6399 and `creditCents` 2000, a balance of 4399, which is the balance of `inv-1`. `creditLiability` has `debitCents` 2000 and `creditCents` 2000, a balance of 0 and side `none`, which is Ada's credit balance.

**Worked example: a payment that overpays.** On `2026-04-05T10:00Z` Ada pays 5000 cents on `inv-1`. The payment `pay-1` has `appliedCents` 4399 and `creditedCents` 601. It posts `jrn-5` (debit `cash`, credit `receivable`, 4399, memo `pay-1`) and then `jrn-6` (debit `cash`, credit `creditLiability`, 601, memo `pay-1`). A payment of exactly 4399 posts only `jrn-5`, because the second amount is 0. Now `receivable` has a balance of 0, `creditLiability` has a credit balance of 601, which is Ada's credit balance, and `cash` has `debitCents` 5000. The trial balance has `totalDebitCents` and `totalCreditCents` of 15399 each.
