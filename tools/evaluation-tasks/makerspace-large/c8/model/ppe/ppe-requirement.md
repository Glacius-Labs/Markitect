---
kind: std/Concept
---
# PPE requirement

The fact that machines of a tool family may be checked in only by members who hold an item. A family can require several items. A requirement is never removed.

**Fields**, in the formats of [[values]]:

- `familyId`: identifier, naming an existing [[equipment/family]]
- `itemId`: identifier, naming an existing [[ppe/ppe-item]]

A record is identified by `familyId` and `itemId` together.
