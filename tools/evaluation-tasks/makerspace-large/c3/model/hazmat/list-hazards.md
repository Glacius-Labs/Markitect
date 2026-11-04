---
kind: Operation
---
# List hazards

Returns all classifications, in ascending `materialId` order. Without any classification the list is empty.

- **Wire name:** `listHazards`, a read.
- **Arguments:** only `actor` and `at`.
- **Result:**
  - `hazards`: a list of [[hazmat/classification]] records
- **Permissions:** member all, staff all, admin all.
