---
kind: std/Concept
---
# Pack product

A product the workshop sells: a block of machine minutes that is valid for a number of days.

**Fields**, in the formats of [[values]]:

- `productId`: identifier
- `name`: text
- `minutes`: integer from `1` to `100000` (the minutes of a pack)
- `priceCents`: integer from `1` to `100000000` cents (the price of a pack)
- `validDays`: integer from `1` to `3650` (the days a pack is valid)

A record is identified by `productId`.
