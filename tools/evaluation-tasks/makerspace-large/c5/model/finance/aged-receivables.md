---
kind: Operation
---
# Aged receivables

Lists the open invoices by how long they have been overdue on a date. An invoice is included when its status is `open` at the request time, whatever the date is. `daysOverdue` of a row is the number of whole days from the date of the invoice's `dueAt` to `date`: the difference of the two calendar dates, and 0 when `date` is on or before the date of `dueAt`. The time of day of `dueAt` does not matter here. `balanceCents` is the invoice's balance at the request time. `rows` is in ascending invoice number, the number `N` of `inv-N` compared as a number.

Each balance falls in one bucket by its `daysOverdue`: `currentCents` for 0 days, `days1to30Cents` for 1 to 30, `days31to60Cents` for 31 to 60 and `over60Cents` for 61 or more. A bucket is the sum of the balances in it and is 0 when it is empty.

Example: an invoice has `dueAt` `2026-04-15T08:00Z`. For `date` `2026-04-15` its `daysOverdue` is 0 (current), for `2026-04-16` it is 1, for `2026-05-15` it is 30, for `2026-05-16` it is 31, for `2026-06-14` it is 60 and for `2026-06-15` it is 61.

- **Wire name:** `agedReceivables`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `date`: date
- **Result:**
  - `date`: date
  - `rows`: a list of [[finance/aged-receivable-row]] records
  - `currentCents`: cent amount
  - `days1to30Cents`: cent amount
  - `days31to60Cents`: cent amount
  - `over60Cents`: cent amount
- **Permissions:** staff all, admin all.
