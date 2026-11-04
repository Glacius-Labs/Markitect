---
kind: std/Concept
---
# Part

A spare part with a unit cost and a stock count; see [[parts/spare-parts]].

**Fields**, in the formats of [[values]]:

- `partId`: identifier
- `name`: text
- `unitCostCents`: integer from `1` to `1000000` cents (the cost of one unit of a part)
- `familyId`: identifier, naming an existing [[equipment/family]] (the family the part fits)
- `stock`: count

A record is identified by `partId`.
