---
kind: Operation
---
# List member packs

Returns all packs of the member, expired and used up ones included, in creation order (ascending pack number). A member without packs gives an empty list.

- **Wire name:** `listMemberPacks`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
- **Result:**
  - `memberId`: identifier
  - `packs`: a list of [[packs/pack]] records
- **Permissions:** member own, staff all, admin all.
