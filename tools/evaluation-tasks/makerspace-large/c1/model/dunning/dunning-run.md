---
kind: std/Rule
---
# Dunning run

A run takes every invoice that is overdue at the request time ([[billing/overdue-invoices]]), in ascending invoice number: the number `N` of `inv-N`, compared as a number. For each invoice it takes the steps that are due, in the order below, all of them in the same run when all are due.

The **days overdue** of an invoice are the whole days from its `dueAt` to the request time, rounded down: the number of complete 1440-minute periods between the two timestamps. An invoice overdue for less than one day has 0 days overdue and no step is due, because every ladder day is at least 1.

A step is due when the days overdue are at least its ladder days and the step has not been taken for this invoice before. A step is taken at most once per invoice, whatever the ladder is later. The steps:

1. **Notice**, at `noticeDays` or more. A notice of kind `dunning` is posted for the invoice's member through [[notices/notices-interface]] with `ref` the invoice identifier and the subject `Invoice <invoiceId> is overdue`.
2. **Fee**, at `feeDays` or more. A late fee [[billing/charge]] is created for the member: `kind` `lateFee`, `ref` the invoice identifier, `quantity` 1, `unitPriceCents` the ladder's `feeCents`, `discountPercent` 0 and `at` the request time. The charge is unbilled and is invoiced like any other; it does not change the overdue invoice.
3. **Suspension**, at `suspendDays` or more. The member's standing becomes `suspended`. When the member is already suspended, the standing does not change, but the step still counts as taken. A notice of kind `suspension` is posted with `ref` the invoice identifier; its subject is the one [[notices/notice-kind]] states.

For each step taken, an action is reported with the invoice identifier and the step (`notice`, `fee` or `suspend`). The invoice's [[dunning/dunning-case|case]] is created in the first run that takes a step for the invoice, and the step's timestamp (`noticedAt`, `feeChargedAt` or `suspendedAt`) is set to the request time.

Paying the invoice later undoes nothing and reinstates nobody. A member who was reinstated is not suspended again for a step already taken. A paid invoice is not overdue and is not visited.

Example: the ladder is `noticeDays` 3, `feeDays` 7, `suspendDays` 14. Invoice `inv-1` is open with `dueAt` `2026-04-15T08:00Z`. A run at `2026-04-18T07:59Z` finds 2 days overdue and takes no step. A run at `2026-04-18T08:00Z` finds 3 days: it posts the notice, creates `dun-1` with `noticedAt` set and reports `inv-1` `notice`. A run at `2026-04-30T08:00Z` finds 15 days: it charges the fee and suspends the member, reporting `inv-1` `fee` and `inv-1` `suspend` in that order. Another run at the same time reports nothing.
