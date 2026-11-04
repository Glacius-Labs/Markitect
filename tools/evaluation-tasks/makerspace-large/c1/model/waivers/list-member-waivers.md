---
kind: Operation
---
# List member waivers

Returns the member's latest signature for each family the member has signed, including signatures of versions that are no longer current. The signatures are ordered by `familyId` ascending.

- **Wire name:** `listMemberWaivers`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
- **Result:**
  - `memberId`: identifier
  - `signatures`: a list of [[waivers/waiver-signature]] records
- **Permissions:** member own, staff all, admin all.
