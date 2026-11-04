---
kind: Operation
---
# List course classes

Returns all classes of the course in every status, with `start` ascending, then in creation order.

- **Wire name:** `listCourseClasses`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `courseId`: identifier, naming an existing [[training/course]]
- **Result:**
  - `courseId`: identifier
  - `classes`: a list of [[classes/class]] records
- **Permissions:** member all, staff all, admin all.
