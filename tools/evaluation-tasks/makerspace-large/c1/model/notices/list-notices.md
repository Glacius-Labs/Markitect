---
kind: Operation
---
# List notices

Returns all notices of the member, acknowledged or not, in creation order.

- **Wire name:** `listNotices`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
- **Result:**
  - `memberId`: identifier
  - `notices`: a list of [[notices/notice]] records
- **Permissions:** member own, staff all, admin all.
