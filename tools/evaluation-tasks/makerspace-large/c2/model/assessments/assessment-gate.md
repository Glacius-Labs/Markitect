---
kind: std/Rule
---
# Assessment gate

At [[training/complete-course]] a member must have passed the assessment of the course, if it has one, in the current enrollment period.

The completion is refused with `assessment_not_passed` when the course has an [[assessments/assessment]] and none of the member's attempts in the current enrollment period has `passed` true ([[assessments/attempts]]). Attempts of earlier periods do not count. A course without an assessment passes the gate.

Example: the pass mark is 70. The member enrolled, scored 65 and then 70 in the first period and completed the course. After a new enrollment the first period's attempts count for nothing: the completion is refused until an attempt of the new period, numbered 1, has a score of 70 or more.
