---
kind: std/Rule
---
# Computed times

Request times and time arguments lie between 2000 and 2099, but some values the program computes from them lie later. Integers grow with the number of requests. This rule and [[integer-limits|integer limits]], [[identifier-limits|identifier limits]] and [[request-limits|request limits]] cover every computed value.

- **Times after 2099.** A computed timestamp or date can lie after 2099. It uses the same format as its type, with a four-digit year, and the Gregorian calendar continues as usual: 2100 is not a leap year, so `2099-03-01` plus 365 days is `2100-03-01`. The latest values a result can show are a `dueAt` of `2100-01-14T23:59Z` (14 days after `2099-12-31T23:59Z`), a loan's `dueOn` of `2100-02-11` (14 days after `2099-12-31`, moved 14 days later by each of two renewals) and a certification's `expiresOn` of `2100-12-31` (365 days after `2099-12-31`). A quote's `validUntil` lies at the latest on `2100-01-30`. A pass's `expiresOn` and an inspection plan's `nextDueOn`, at most 3650 days after a date of 2099, lie at the latest on `2109-12-29`. Points used only in comparisons can also lie after 2099: the first minute after the month `2099-12` and after the date `2099-12-31` are both `2100-01-01T00:00Z`. The lapse date of a waiver signature is used only in comparisons and lies at the latest on `2100-12-31`. No computed time lies before `2000-01-01T00:00Z`.
- **The end of the calendar.** No request time is later than `2099-12-31T23:59Z`. The following consequences follow from the rules of this model and are stated so that they are read only one way:
  - `2099-11` is the last month that can be invoiced. `generateInvoices` for `2099-12` is never accepted: when it reaches the operation rules (check 7), it is refused with `month_not_ended`. The revenue report for `2099-12` always has `invoicesGenerated` `false`.
  - An invoice whose `dueAt` lies after 2099 and a loan whose `dueOn` lies after 2099 are never overdue. A certification whose `expiresOn` lies after 2099 is never reported `expired`. A waiver signature made in 2099 never lapses at a request time.
  - The last day a utilization report can include is `2099-12-30`, because `to` is at most `2099-12-31` and is excluded from the range.
  - A maintenance window ends at the latest at `2099-12-31T23:59Z`, so the last minute of 2099 cannot be covered. Reservations end by closing time and are not affected.
