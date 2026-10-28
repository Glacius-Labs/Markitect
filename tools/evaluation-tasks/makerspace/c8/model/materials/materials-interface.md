---
kind: Interface
---
# Materials interface

What other modules may use of materials: its operations and the material and purchase records.

- **Operations:** [[materials/add-material]], [[materials/restock-material]], [[materials/purchase-material]], [[materials/list-materials]].
- **Uses:** [[members/members-interface]], [[billing/billing-interface]].
- **Records:** [[materials/material]], [[materials/purchase]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `insufficient_stock` | The material's stock is lower than the requested quantity. |
