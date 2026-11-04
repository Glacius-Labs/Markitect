---
kind: Interface
---
# Deposits interface

What other modules may use of deposits: its operations, the deposit record and the refusal `deposit_unpaid`. No other module asks anything else of it.

- **Operations:** [[deposits/take-deposit]], [[deposits/release-deposit]], [[deposits/forfeit-deposit]], [[deposits/list-deposits]].
- **Uses:** [[members/members-interface]], [[billing/billing-interface]], [[ledger/ledger-interface]].
- **Records:** [[deposits/deposit]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `deposit_unpaid` | The deposit's charge is not on an invoice with status `paid`. |
