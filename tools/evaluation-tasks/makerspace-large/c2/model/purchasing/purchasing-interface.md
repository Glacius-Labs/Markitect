---
kind: Interface
---
# Purchasing interface

What other modules may use of purchasing: its operations, the supplier, order and order line records and the refusal codes below. Other modules ask purchasing for nothing else.

- **Operations:** [[purchasing/add-supplier]], [[purchasing/set-supplier-status]], [[purchasing/create-order]], [[purchasing/add-order-line]], [[purchasing/submit-order]], [[purchasing/approve-order]], [[purchasing/receive-order]], [[purchasing/list-orders]].
- **Uses:** [[materials/materials-interface]], [[ledger/ledger-interface]].
- **Records:** [[purchasing/supplier]], [[purchasing/order]], [[purchasing/order-line]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `approval_requires_admin` | The order's total is above 100000 cents and the actor is staff. |
| `duplicate_line` | The order already has a line for the material. |
| `empty_order` | The order has no line. |
| `not_on_order` | The order has no line for the material. |
| `over_receipt` | The receipt would bring the line's received quantity above its ordered quantity. |
| `self_approval` | The actor created the order. |
| `supplier_blocked` | The supplier is blocked. |

`receiveOrder` also answers `stocktake_open`, which stocktake interface states. Purchasing asks materials whether the material has a count in an open stocktake.
