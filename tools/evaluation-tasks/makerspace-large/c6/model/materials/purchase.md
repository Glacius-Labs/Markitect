---
kind: std/Concept
---
# Purchase

A sale of material to a member, generated identifier `pur-N`.

**Fields**, in the formats of [[values]]:

- `purchaseId`: identifier
- `memberId`: identifier, naming an existing [[members/member]]
- `materialId`: identifier, naming an existing [[materials/material]]
- `quantity`: integer from `1` to `10000` (the number of units bought in one request)
- `unitPriceCents`: integer from `1` to `1000000` cents (the price of one unit of a material)
- `discountPercent`: percent
- `amountCents`: cent amount
- `at`: timestamp

A record is identified by `purchaseId`.
