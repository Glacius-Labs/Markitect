---
kind: Operation
---
# Enroll in course

Enrolls the member in the course. The enrollment gets status `enrolled`, `enrolledAt` equal to the request time and `completedAt` `null`. A course [[billing/charge]] is created for the member: `kind` `course`, `ref` the course identifier, `quantity` 1, `unitPriceCents` the course fee, `discountPercent` the percent that [[scholarships/course-discount]] decides (0 when neither a scholarship award nor a redeemed promotion applies), `at` the request time. A course with fee 0 creates a charge of 0 cents. The `course` result is the course after the change.

Rules of other modules that apply here: [[billing/charges-and-invoices]], [[members/membership]], [[guardians/guardian-consent]], [[scholarships/course-discount]] and [[promotions/redeemed-promotions]].

- **Wire name:** `enrollInCourse`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `courseId`: identifier, naming an existing [[training/course]]
  - `memberId`: identifier, naming an existing [[members/member]]
- **Refusals** at check 7, in this order:
  1. `member_suspended` when the member is not in good standing ([[members/members-interface]]).
  2. `guardian_consent_missing` when the member is a minor on the date of the request time and has no consent for the course's family ([[guardians/guardian-consent]]).
  3. `already_enrolled` when the member's enrollment in this course has status `enrolled`.
  4. `course_full` when `enrolledCount` equals `capacity`.
- **Result:**
  - `enrollment`: a [[training/enrollment]] record
  - `course`: a [[training/course]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `courseId/memberId`.
