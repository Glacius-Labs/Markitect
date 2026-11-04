---
kind: Interface
---
# Parts interface

What other modules may use of parts: its operations and the part record. Other modules also ask for a part by its identifier, with its family, unit cost and stock, and to decrease a part's stock. Parts adds no refusal codes.

- **Operations:** [[parts/add-part]], [[parts/restock-part]], [[parts/list-parts]].
- **Uses:** [[equipment/equipment-interface]].
- **Records:** [[parts/part]].
