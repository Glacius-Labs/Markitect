---
kind: Operation
---
# List member loans

Returns all loans of the member, active and returned, in creation order.

- **Wire name:** `listMemberLoans`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
- **Result:**
  - `memberId`: identifier
  - `loans`: a list of [[loans/loan]] records
- **Permissions:** member own, staff all, admin all.
