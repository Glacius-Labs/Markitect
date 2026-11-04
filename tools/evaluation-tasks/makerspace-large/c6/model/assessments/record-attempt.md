---
kind: Operation
---
# Record attempt

Records an attempt of the member in the current enrollment period ([[assessments/attempts]]): `number` is one more than the attempts of that period so far, `score` is as supplied, `passed` is true when `score` is at least the `passMark` of the course's assessment, and `at` is the request time.

Example: with `passMark` 70 and `maxAttempts` 2, a score of 69 is not passed and a score of 70 is passed. After two attempts a third is refused with `attempts_exhausted`, even when one of them passed. After a first attempt that passed, a second is still accepted.

- **Wire name:** `recordAttempt`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `courseId`: identifier, naming an existing [[training/course]]
  - `memberId`: identifier, naming an existing [[members/member]]
  - `score`: integer from `0` to `100` (the score of the attempt)
- **Refusals** at check 7, in this order:
  1. `no_assessment` when the course has no assessment.
  2. `not_enrolled` when the member has no enrollment in the course with status `enrolled` ([[training/training-interface]]).
  3. `attempts_exhausted` when the attempts of the member's current enrollment period equal the `maxAttempts` of the assessment.
- **Result:**
  - `attempt`: a [[assessments/attempt]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `courseId/memberId`.
