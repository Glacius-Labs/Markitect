---
kind: std/Rule
---
# Late-fee charge

A tool that is late costs money per day. The charge for the days of late fee not yet charged on a [[loans/loan]] is created by [[loans/return-tool]] and by [[loans/charge-overdue-loans]]. It is a [[billing/charge]] for the member of the loan: `kind` `lateFee`, `ref` the loan identifier, `quantity` the days overdue less the loan's `chargedDays`, `unitPriceCents` the tool's `lateFeeCentsPerDay`, `discountPercent` 0 and `at` the request time. The amount follows [[money]]. The loan's `chargedDays` then becomes the days overdue ([[loans/overdue-loans]]), so a day is never charged twice. No charge is created when the days overdue are not more than `chargedDays`.

The charge is unbilled and is invoiced like any other ([[billing/charges-and-invoices]]).

Example: a loan with `borrowedAt` `2026-03-02T10:00Z` of a tool with a fee of 200 cents has `dueOn` `2026-03-16`. It is not overdue on `2026-03-16`. A charging run on `2026-03-19` charges 3 days, 600 cents. A run on `2026-03-20` charges 1 day, 200 cents. A return on `2026-03-22` charges the remaining 2 days, 400 cents.
