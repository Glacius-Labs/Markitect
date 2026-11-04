---
kind: Interface
---
# Materials interface

What other modules may use of materials: its operations and the material and purchase records. Other modules also ask materials for:

- a material's existence, its current unit price and its current stock;
- the units a member bought of a material in a month: the sum of `quantity` over the member's purchases of that material whose `at` lies in that month;
- a purchase's member and amount: for a purchase identifier, the purchase's `memberId` and `amountCents`;
- adding received stock: the stock of a material increases by a quantity;
- setting stock from a count: the stock of a material becomes a given number of units;
- drawing stock for consumables: for a material and a number of units, the stock decreases by the smaller of the units and the stock, and the number drawn is answered; it may be 0;
- whether a material has a count in an open stocktake. Materials answers this from the answer of stocktake that it needs, as below.

Materials in turn needs two answers from modules that depend on it. From hazmat it needs, for a member, a material, a quantity and the request time, which refusal of the hazard gate applies to a purchase, if any. From stocktake it needs whether a material has a count in an open stocktake. It declares these needs, and the protocol module supplies the answers from hazmat interface and stocktake interface.

- **Operations:** [[materials/add-material]], [[materials/restock-material]], [[materials/purchase-material]], [[materials/list-materials]].
- **Uses:** [[members/members-interface]], [[billing/billing-interface]], [[promotions/promotions-interface]].
- **Records:** [[materials/material]], [[materials/purchase]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `insufficient_stock` | The material's stock is lower than the requested quantity. |

`purchaseMaterial` also answers `hazard_not_certified` and `hazard_limit_exceeded`, which hazmat interface states, and `stocktake_open`, which stocktake interface states. `restockMaterial` answers `stocktake_open`.
