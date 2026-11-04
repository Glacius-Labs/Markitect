---
kind: Operation
---
# Define assessment

Creates the assessment of the course. It applies at once to members who are already enrolled: their completion needs a passed attempt, and no attempt was possible before this.

- **Wire name:** `defineAssessment`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `courseId`: identifier, naming an existing [[training/course]]
  - `passMark`: integer from `0` to `100` (the lowest score that passes)
  - `maxAttempts`: integer from `1` to `10` (the number of attempts allowed in one enrollment period)
- **Refusals** at check 7, in this order:
  1. `already_exists` when the course has an assessment.
- **Result:**
  - `assessment`: a [[assessments/assessment]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `courseId`.
