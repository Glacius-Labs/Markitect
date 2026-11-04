---
kind: Operation
---
# List member promotions

Returns all redemptions of the member, used and unused, in the order they were made (creation order). A member without redemptions gives an empty list.

- **Wire name:** `listMemberPromotions`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
- **Result:**
  - `memberId`: identifier
  - `redemptions`: a list of [[promotions/redemption]] records
- **Permissions:** member own, staff all, admin all.
