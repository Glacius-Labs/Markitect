---
kind: Operation
---
# List awards

Returns all awards of the member, active and revoked, in creation order (ascending award number). A member without awards gives an empty list.

- **Wire name:** `listAwards`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
- **Result:**
  - `memberId`: identifier
  - `awards`: a list of [[scholarships/award]] records
- **Permissions:** member own, staff all, admin all.
