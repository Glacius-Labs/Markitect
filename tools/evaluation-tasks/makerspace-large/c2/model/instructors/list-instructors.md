---
kind: Operation
---
# List instructors

Returns the qualifications for the family, in ascending `userId` order.

- **Wire name:** `listInstructors`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `familyId`: identifier, naming an existing [[equipment/family]]
- **Result:**
  - `familyId`: identifier
  - `instructors`: a list of [[instructors/qualification]] records
- **Permissions:** member all, staff all, admin all.
