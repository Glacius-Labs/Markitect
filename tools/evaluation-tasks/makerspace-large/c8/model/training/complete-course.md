---
kind: Operation
---
# Complete course

Records that the member completed the course: the enrollment gets status `completed` and `completedAt` equal to the request time. The member's certification for the course's family is issued as [[training/certifications]] states. Standing is not checked.

Rules of other modules that apply here: attendance gate and assessment gate.

- **Wire name:** `completeCourse`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `courseId`: identifier, naming an existing [[training/course]]
  - `memberId`: identifier, naming an existing [[members/member]]
- **Refusals** at check 7, in this order:
  1. `not_enrolled` when the member has no enrollment in this course with status `enrolled`.
  2. `attendance_missing` when a class of the course requires the member's attendance and the member did not attend it (attendance gate).
  3. `assessment_not_passed` when the course has an assessment and the member has not passed it in the current enrollment period (assessment gate).
- **Result:**
  - `enrollment`: a [[training/enrollment]] record
  - `certification`: a [[training/certification]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `courseId/memberId`.
