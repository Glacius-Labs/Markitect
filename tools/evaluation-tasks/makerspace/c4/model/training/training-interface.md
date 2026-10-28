---
kind: Interface
---
# Training interface

What other modules may use of training: its operations, the course, enrollment and certification records, and the refusal `not_certified`. Other modules also ask whether a member holds a certification for a family that is valid on a date.

- **Operations:** [[training/add-course]], [[training/enroll-in-course]], [[training/complete-course]], [[training/revoke-certification]], [[training/list-certifications]].
- **Uses:** [[members/members-interface]], [[equipment/equipment-interface]], [[billing/billing-interface]].
- **Records:** [[training/course]], [[training/enrollment]], [[training/certification]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_enrolled` | The member is already enrolled in the course. |
| `course_full` | The course has no free place. |
| `not_certified` | The member has no certification valid on the relevant date for the machine's family, or there is no certification to revoke. |
| `not_enrolled` | The member is not enrolled in the course. |
