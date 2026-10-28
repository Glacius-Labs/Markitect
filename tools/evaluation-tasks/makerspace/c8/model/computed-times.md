---
kind: std/Rule
---
# Computed times

Request times and time arguments lie between 2000 and 2099, but some values the program computes from them lie later. Integers grow with the number of requests. This rule and [[integer-limits|integer limits]], [[identifier-limits|identifier limits]] and [[request-limits|request limits]] cover every computed value.

- **Times after 2099.** A computed timestamp or date can lie after 2099. It uses the same format as its type, with a four-digit year, and the Gregorian calendar continues as usual: 2100 is not a leap year, so `2099-03-01` plus 365 days is `2100-03-01`. The latest value a result can show is a `dueAt` of `2100-01-14T23:59Z` (14 days after `2099-12-31T23:59Z`). Points used only in comparisons can also lie after 2099: the first minute after the month `2099-12` and after the date `2099-12-31` are both `2100-01-01T00:00Z`, and the block compares the request time with `dueAt` plus 43200 minutes, a point as late as `2100-02-13T23:59Z`. No computed time lies before `2000-01-01T00:00Z`.
- **The end of the calendar.** No request time is later than `2099-12-31T23:59Z`. The following consequences follow from the rules of this model and are stated so that they are read only one way:
  - `2099-11` is the last month that can be invoiced. `generateInvoices` for `2099-12` is never accepted: when it reaches the operation rules (check 7), it is refused with `month_not_ended`. The revenue report for `2099-12` always has `invoicesGenerated` `false`.
  - An invoice whose `dueAt` lies after 2099 is never overdue.
  - An invoice whose `dueAt` is `2099-12-01T23:59Z` or later can never block.
  - The last day a utilization report can include is `2099-12-30`, because `to` is at most `2099-12-31` and is excluded from the range.
  - A maintenance window or a lockout ends at the latest at `2099-12-31T23:59Z`, so the last minute of 2099 cannot be covered. Reservations end by closing time and are not affected.
