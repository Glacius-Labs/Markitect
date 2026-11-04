---
kind: Operation
---
# Promote waitlist

Fills the free places of the course from its waiting list as [[waitlists/waitlist-promotion]] states. `enrollments` are the enrollments made, in join order; `skipped` are the entries that were refused, in join order. Both are empty lists when nothing happened. A course with no free place, or with no entries, is not an error.

Rules of other modules that apply here: [[training/enroll-in-course]] and, for the notice, [[notices/notices-interface]].

- **Wire name:** `promoteWaitlist`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `courseId`: identifier, naming an existing [[training/course]]
- **Result:**
  - `courseId`: identifier
  - `enrollments`: a list of [[training/enrollment]] records
  - `skipped`: a list of [[waitlists/skipped-entry]] records
- **Permissions:** staff all, admin all.
- **Audit target:** `courseId`.
