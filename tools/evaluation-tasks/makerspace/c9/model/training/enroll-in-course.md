---
kind: Operation
---
# Enroll in course

Enrolls the member in the course. The enrollment gets status `enrolled`, `enrolledAt` equal to the request time and `completedAt` `null`. A course [[billing/charge]] is created for the member: `kind` `course`, `ref` the course identifier, `quantity` 1, `unitPriceCents` the course fee, `discountPercent` 0, `at` the request time. A course with fee 0 creates a charge of 0 cents. The `course` result is the course after the change.

Rules of other modules that apply here: [[billing/charges]] and [[members/membership]].

- **Wire name:** `enrollInCourse`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `courseId`: identifier, naming an existing [[training/course]]
  - `memberId`: identifier, naming an existing [[members/member]]
- **Refusals** at check 7, in this order:
  1. `member_suspended` when the member is not in good standing ([[members/members-interface]]).
  2. `already_enrolled` when the member's enrollment in this course has status `enrolled`.
  3. `course_full` when `enrolledCount` equals `capacity`.
- **Result:**
  - `enrollment`: a [[training/enrollment]] record
  - `course`: a [[training/course]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `courseId/memberId`.
