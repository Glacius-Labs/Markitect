---
kind: Interface
---
# Lockers interface

What other modules may use of lockers: its operations, the locker and rental records and the refusals below. No other module asks anything else of it.

- **Operations:** [[lockers/add-locker]], [[lockers/rent-locker]], [[lockers/end-locker-rental]], [[lockers/bill-lockers]].
- **Uses:** [[members/members-interface]], [[billing/billing-interface]].
- **Records:** [[lockers/locker]], [[lockers/rental]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_billed` | The lockers of the month have already been billed. |
| `locker_taken` | The locker has an active rental. |
| `not_current_month` | The month is not the month of the request time. |
| `rental_limit` | The member already has 2 active rentals. |
