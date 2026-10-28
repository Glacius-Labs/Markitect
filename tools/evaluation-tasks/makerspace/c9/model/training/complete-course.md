---
kind: Operation
---
# Complete course

Records that the member completed the course: the enrollment gets status `completed` and `completedAt` equal to the request time. The member's certification for the course's family is issued as [[training/certifications]] states. Standing is not checked.

- **Wire name:** `completeCourse`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `courseId`: identifier, naming an existing [[training/course]]
  - `memberId`: identifier, naming an existing [[members/member]]
- **Refusals** at check 7, in this order:
  1. `not_enrolled` when the member has no enrollment in this course with status `enrolled`.
- **Result:**
  - `enrollment`: a [[training/enrollment]] record
  - `certification`: a [[training/certification]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `courseId/memberId`.
