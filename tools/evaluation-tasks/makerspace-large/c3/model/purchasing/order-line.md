---
kind: std/Concept
---
# Order line

One material on an [[purchasing/order]]. An order has at most one line per material.

**Fields**, in the formats of [[values]]:

- `materialId`: identifier, naming an existing [[materials/material]]
- `quantity`: integer from `1` to `100000` (the number of units ordered)
- `unitCostCents`: integer from `1` to `1000000` cents (the cost of one unit from the supplier; it is not the material's unit price)
- `receivedQuantity`: count, the number of units received so far; it starts at 0 and is never above `quantity`
