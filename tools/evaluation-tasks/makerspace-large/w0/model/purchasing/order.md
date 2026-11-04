---
kind: std/Concept
---
# Order

An order of materials from one supplier, generated identifier `po-N`. It moves from `draft` to `submitted` to `approved` to `received`.

**Fields**, in the formats of [[values]]:

- `orderId`: identifier
- `supplierId`: identifier, naming an existing [[purchasing/supplier]]
- `status`: an [[purchasing/order-status]]: `draft`, `submitted`, `approved` or `received`
- `createdBy`: identifier, the user who created the order
- `totalCents`: cent amount, the sum over the lines of `quantity × unitCostCents`; 0 while there is no line
- `lines`: a list of [[purchasing/order-line]] records, in the order they were added

A record is identified by `orderId`.
