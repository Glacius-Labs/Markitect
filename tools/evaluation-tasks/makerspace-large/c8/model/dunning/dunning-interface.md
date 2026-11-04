---
kind: Interface
---
# Dunning interface

What other modules may use of dunning: its operations, the ladder, case and action records, and the refusals below. No other module asks anything else of it.

- **Operations:** [[dunning/set-dunning-ladder]], [[dunning/run-dunning]], [[dunning/list-dunning-cases]].
- **Uses:** [[members/members-interface]], [[billing/billing-interface]], [[notices/notices-interface]].
- **Records:** [[dunning/dunning-ladder]], [[dunning/dunning-case]], [[dunning/dunning-action]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `invalid_ladder` | The days of a ladder are not strictly increasing from notice to fee to suspension. |
| `no_ladder` | No ladder has been set. |
