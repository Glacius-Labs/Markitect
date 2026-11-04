---
kind: std/Concept
---
# Fund

A budget from which course discounts of awards are paid. `spentCents` is the sum of the discounts given so far and starts at 0.

**Fields**, in the formats of [[values]]:

- `fundId`: identifier
- `budgetCents`: integer from `1` to `100000000` cents (the budget of a fund)
- `spentCents`: cent amount

A record is identified by `fundId`.
