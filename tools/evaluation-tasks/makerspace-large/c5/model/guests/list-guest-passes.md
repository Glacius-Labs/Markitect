---
kind: Operation
---
# List guest passes

Returns all guest passes of the member, revoked or not, in creation order.

- **Wire name:** `listGuestPasses`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
- **Result:**
  - `memberId`: identifier
  - `guestPasses`: a list of [[guests/guest-pass]] records
- **Permissions:** member own, staff all, admin all.
