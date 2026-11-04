---
kind: std/Concept
---
# Material

A material sold by unit, with a unit price and a stock count.

**Fields**, in the formats of [[values]]:

- `materialId`: identifier
- `name`: text
- `unitPriceCents`: integer from `1` to `1000000` cents (the price of one unit of a material)
- `stock`: count

A record is identified by `materialId`.
