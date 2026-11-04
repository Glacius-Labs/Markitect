---
kind: Operation
---
# Issue PPE

The member holds the item. The item's stock decreases by 1. The `item` result is the item after the change.

Rules of other modules that apply here: [[members/membership]].

- **Wire name:** `issuePpe`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
  - `itemId`: identifier, naming an existing [[ppe/ppe-item]]
- **Refusals** at check 7, in this order:
  1. `member_suspended` when the member is not in good standing ([[members/members-interface]]).
  2. `already_issued` when the member holds the item.
  3. `out_of_stock` when the item's stock is 0.
- **Result:**
  - `item`: a [[ppe/ppe-item]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `memberId/itemId`.
