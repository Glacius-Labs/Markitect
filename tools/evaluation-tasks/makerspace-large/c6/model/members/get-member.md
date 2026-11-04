---
kind: Operation
---
# Get member

Returns the member.

- **Wire name:** `getMember`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
- **Result:**
  - `member`: a [[members/member]] record
- **Permissions:** member own, staff all, admin all.
