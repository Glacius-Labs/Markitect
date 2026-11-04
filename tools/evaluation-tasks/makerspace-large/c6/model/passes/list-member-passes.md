---
kind: Operation
---
# List member passes

Returns all passes of the member, expired and used up ones included, in creation order (ascending pass number). A member without passes gives an empty list.

- **Wire name:** `listMemberPasses`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
- **Result:**
  - `memberId`: identifier
  - `passes`: a list of [[passes/pass]] records
- **Permissions:** member own, staff all, admin all.
