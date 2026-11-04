---
kind: std/Rule
internal: true
---
# Volunteer credit

The **rate** is a number of cents per hour from `0` to `100000`. It starts at 0 and is changed only by [[volunteering/set-volunteer-rate]]. An entry is paid at the rate in force when it is approved; a later change does not touch decided entries.

The credit of an approved entry is `creditCents = (minutes × centsPerHour + 30) div 60`, which is the exact value rounded half up to a whole cent. It is added to the member's credit balance through billing, which posts it as [[ledger/postings]] states. With a rate of 0 the credit is 0, the entry is still approved, and nothing is posted.

The **cap**: the minutes of a member's entries with status `approved` and a `date` in a month are at most 1200. An approval is refused when the approved minutes of that member in the month of the entry's `date`, plus the entry's `minutes`, would be above 1200; reaching exactly 1200 is allowed. Pending and rejected entries do not count. The month is the month of the entry's `date`, not of the request time.

Example: at 1500 cents per hour, 100 minutes give `(100 × 1500 + 30) div 60` = 2500 cents. At 1 cent per hour, 30 minutes give `(30 × 1 + 30) div 60` = 1 cent. A member with 1100 approved minutes in June can have a 100-minute entry of June approved, and then no more.
