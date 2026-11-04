---
kind: Operation
---
# List deposits

Returns all of the member's deposits, in creation order (ascending deposit number). A member without deposits gives an empty list.

- **Wire name:** `listDeposits`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
- **Result:**
  - `memberId`: identifier
  - `deposits`: a list of [[deposits/deposit]] records
- **Permissions:** member own, staff all, admin all.
