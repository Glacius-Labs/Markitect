---
kind: std/Concept
---
# Classification

The hazard classification of one material: the tool family whose certification a member needs to buy it, and the most units one member may buy of it in a month. A material has at most one classification.

**Fields**, in the formats of [[values]]:

- `materialId`: identifier, naming an existing [[materials/material]]
- `familyId`: identifier, naming an existing [[equipment/family]]
- `monthlyLimit`: integer from `1` to `10000` (the most units of the material one member may buy in a month)

A record is identified by `materialId`.
