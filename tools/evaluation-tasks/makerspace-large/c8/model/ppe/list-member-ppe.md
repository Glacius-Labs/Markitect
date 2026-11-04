---
kind: Operation
---
# List member PPE

Returns the items the member holds, with their current stock, ordered by `itemId` ascending.

- **Wire name:** `listMemberPpe`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
- **Result:**
  - `memberId`: identifier
  - `items`: a list of [[ppe/ppe-item]] records
- **Permissions:** member own, staff all, admin all.
