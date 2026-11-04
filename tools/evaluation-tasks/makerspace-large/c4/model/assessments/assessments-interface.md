---
kind: Interface
---
# Assessments interface

What other modules may use of assessments: its operations and the assessment and attempt records. Other modules also ask whether a member has met the assessment requirement of a course.

- **Operations:** [[assessments/define-assessment]], [[assessments/record-attempt]], [[assessments/list-attempts]].
- **Uses:** [[members/members-interface]], [[training/training-interface]].
- **Records:** [[assessments/assessment]], [[assessments/attempt]].

What other modules ask of assessments that is not an operation:

- Whether a member has met the assessment requirement of a course, given the course and the member: yes when [[assessments/assessment-gate]] passes. Assessments learns the member's current enrollment period from training.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `assessment_not_passed` | The course has an assessment and the member has not passed it in the current enrollment period. |
| `attempts_exhausted` | The member has used all attempts of the current enrollment period. |
| `no_assessment` | The course has no assessment. |
