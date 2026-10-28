---
kind: std/Rule
---
# Loan policy

The loan policy states the admission limit, the calendar-day durations and the renewal limit used for loans. It is the single source of these values, each a required positive integer:

| Value | Setting | Meaning |
| --- | --- | --- |
| Loan limit | `3` | The maximum resulting count of currently open loans per member when admitting a new loan. It is an admission limit, not permission to remove existing loans when they exceed a later lower limit. |
| Loan period | `14` days | The number of calendar days added to the borrowing date to determine a new loan's due date for a standard copy. Weekends and holidays do not alter this interval. It does not describe a time of day or a scheduled job. |
| Short loan period | `7` days | The number of calendar days added to the borrowing date to determine a new loan's due date for a short copy. Weekends and holidays do not alter this interval. It does not describe a time of day or a scheduled job. |
| Renewal limit | `1` | How many times one loan can be renewed. Each renewal moves its due date forward once. |

For [[lending/borrow]], the resulting current open-loan count of the member must not exceed the loan limit. Count every open [[lending/loan]], including overdue loans, and exclude closed loans. An overdue loan adds no separate borrowing restriction. A new loan's due date is its borrowing date plus the loan period for a standard [[catalog/copy]], or plus the short loan period for a short copy.

A loan can be renewed at most as often as the renewal limit allows; each [[lending/renew]] replaces the loan's due date with the later one, which is then the loan's current due date.

[[lending/return]] remains possible regardless of the current limit or overdue state. The values do not authorize deletion or forced return of existing loans: a lower limit neither deletes existing loans nor forces their return; while at or above that limit, the member cannot start another loan.
