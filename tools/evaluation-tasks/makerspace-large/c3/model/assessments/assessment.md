---
kind: std/Concept
---
# Assessment

The test a member must pass to complete a course: the lowest passing score and the number of attempts allowed in one enrollment period. A course has at most one assessment, and it is never changed or removed.

**Fields**, in the formats of [[values]]:

- `courseId`: identifier, naming an existing [[training/course]]
- `passMark`: integer from `0` to `100` (the lowest score that passes)
- `maxAttempts`: integer from `1` to `10` (the number of attempts allowed in one enrollment period)

A record is identified by `courseId`.
