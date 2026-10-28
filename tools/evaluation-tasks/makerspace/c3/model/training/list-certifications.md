---
kind: Operation
---
# List certifications

Returns all of the member's certifications, including expired and revoked ones.

- **Wire name:** `listCertifications`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
- **Result:**
  - `memberId`: identifier
  - `certifications`: a list of [[training/certification]] records
- **Permissions:** member own, staff all, admin all.
