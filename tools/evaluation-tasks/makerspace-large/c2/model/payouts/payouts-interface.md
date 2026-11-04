---
kind: Interface
---
# Payouts interface

What other modules may use of payouts: its operations and the instructor rate and payout records. No other module asks anything else of payouts.

- **Operations:** [[payouts/set-instructor-rate]], [[payouts/generate-payouts]], [[payouts/settle-payout]], [[payouts/list-payouts]].
- **Uses:** [[members/members-interface]], [[ledger/ledger-interface]], [[instructors/instructors-interface]], [[classes/classes-interface]].
- **Records:** [[payouts/instructor-rate]], [[payouts/payout]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_generated` | Payouts for the month have already been generated. |

[[payouts/generate-payouts]] also uses the code `month_not_ended` that invoicing uses. Here it means that the month has not ended at the request time.
