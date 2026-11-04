---
kind: std/Rule
---
# Stocktake freeze

A material is **counted** when it has a [[stocktake/count]] in an open [[stocktake/stocktake]]. A counted material is frozen: three operations refuse to change its stock with `stocktake_open`, in their own order of refusals:

- [[materials/purchase-material]],
- [[materials/restock-material]],
- `receiveOrder`.

A material without a count is not frozen, even while a stocktake is open. Recording a count freezes the material at once; replacing the count keeps it frozen. Closing the stocktake ends the freeze of all its materials. A closed stocktake freezes nothing.

The freeze stops nothing else. In particular the stock that consumables draw at check-out still decreases, and closing compares each `counted` with the stock at that moment.
