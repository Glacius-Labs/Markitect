---
kind: std/Rule
internal: true
---
# Attempts

An **enrollment period** of a member in a course starts with an enrollment and ends with its completion. The first enrollment of the member in the course starts period 1. Every later enrollment, which can only follow a completion, starts the next period, even when it is made at the same time as that completion. The number of a period is the number of the member's enrollments in the course so far; training answers it ([[training/training-interface]]).

An [[assessments/attempt]] belongs to the period in which it was recorded. Its `number` is the count of the attempts of that period so far plus one, so the numbers of a period count 1, 2, 3, and a new period starts again at 1. The attempts of earlier periods do not count in a later one.

An attempt is `passed` when its `score` is at least the `passMark` of the course's [[assessments/assessment]]. A `passMark` of 0 is passed by every score. Attempts are never changed or removed.
