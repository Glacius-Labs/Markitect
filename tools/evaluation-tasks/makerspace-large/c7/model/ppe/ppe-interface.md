---
kind: Interface
---
# PPE interface

What other modules may use of ppe: its operations, the item and requirement records and the refusal codes below. Other modules also ask which items a family requires, and whether a member holds an item. The rule is [[ppe/ppe-gate]].

- **Operations:** [[ppe/add-ppe-item]], [[ppe/require-ppe]], [[ppe/issue-ppe]], [[ppe/return-ppe]], [[ppe/list-member-ppe]].
- **Uses:** [[members/members-interface]], [[equipment/equipment-interface]].
- **Records:** [[ppe/ppe-item]], [[ppe/ppe-requirement]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_issued` | The member already holds the item. |
| `not_issued` | The member does not hold the item. |
| `out_of_stock` | The item's stock is 0. |
| `ppe_missing` | The member does not hold every item the machine's family requires. |
