---
kind: Operation
---
# Return PPE

The member no longer holds the item. The item's stock increases by 1. The `item` result is the item after the change. Standing is not checked.

- **Wire name:** `returnPpe`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
  - `itemId`: identifier, naming an existing [[ppe/ppe-item]]
- **Refusals** at check 7, in this order:
  1. `not_issued` when the member does not hold the item.
- **Result:**
  - `item`: a [[ppe/ppe-item]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `memberId/itemId`.
