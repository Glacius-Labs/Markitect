---
kind: Interface
---
# Training interface

What other modules may use of training: its operations, the course, enrollment and certification records, and the refusal `not_certified`. Other modules also ask whether a member holds a certification for a family that is valid on a date, for the family of a course, for a member's enrollment in a course, for the free places of a course, and to enroll a member in a course. Training in turn needs to know whether a member still owes attendance for a course and whether a member has passed the assessment of a course; it declares these needs, and the protocol module supplies the answers from classes and assessments.

- **Operations:** [[training/add-course]], [[training/enroll-in-course]], [[training/complete-course]], [[training/revoke-certification]], [[training/list-certifications]].
- **Uses:** [[members/members-interface]], [[equipment/equipment-interface]], [[billing/billing-interface]], [[guardians/guardians-interface]], [[promotions/promotions-interface]], [[scholarships/scholarships-interface]].
- **Records:** [[training/course]], [[training/enrollment]], [[training/certification]].

What other modules ask of training that is not an operation:

- Whether a member holds a certification for a family that is valid on a date, as [[training/certifications]] states.
- The family of a course.
- A member's enrollment in a course: whether one exists and, when it does, its `status` and `enrolledAt`, and the number of times the member has enrolled in the course so far. The first enrollment is number 1, and each later enrollment, which reuses and resets the record after a completion, counts once more.
- The free places of a course: its `capacity` less its `enrolledCount`.
- Enrolling a member in a course at a request time, exactly as [[training/enroll-in-course]] does. It makes the refusals of that operation in their order, from `member_suspended` to `course_full`. When none applies it makes every effect of the operation, the enrollment, the charge and the discounts included, and answers the enrollment. When one applies it answers the refusal code and changes nothing. It is not an operation: the checks of roles and ownership and the audit entry belong to the operation that asks.

What training needs from other modules, which the protocol module supplies:

- From classes: whether a member still owes attendance for a course, given the course, the member, the `enrolledAt` of the member's enrollment and the request time (attendance gate).
- From assessments: whether a member has met the assessment requirement of a course, given the course and the member (assessment gate).

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_enrolled` | The member is already enrolled in the course. |
| `course_full` | The course has no free place. |
| `not_certified` | The member has no certification valid on the relevant date for the machine's family, or there is no certification to revoke. |
| `not_enrolled` | The member is not enrolled in the course. Where an operation of another module names a different condition, that operation states it. |

Training's operations also refuse with `guardian_consent_missing` ([[guardians/guardians-interface]]), `attendance_missing` (classes interface) and `assessment_not_passed` (assessments interface), whose meanings those interfaces state.
