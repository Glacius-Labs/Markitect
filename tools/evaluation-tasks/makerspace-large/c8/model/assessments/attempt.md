---
kind: std/Concept
---
# Attempt

One attempt of a member at the assessment of a course. An attempt has no identifier of its own.

**Fields**, in the formats of [[values]]:

- `courseId`: identifier, naming an existing [[training/course]]
- `memberId`: identifier, naming an existing [[members/member]]
- `number`: count, 1 for the first attempt of an enrollment period ([[assessments/attempts]])
- `score`: integer from `0` to `100` (the score of the attempt)
- `passed`: flag
- `at`: timestamp
