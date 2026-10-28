---
kind: std/Concept
---
# Course

A course that teaches the safe use of one tool family, with a fee and a capacity.

**Fields**, in the formats of [[values]]:

- `courseId`: identifier
- `familyId`: identifier, naming an existing [[equipment/family]]
- `title`: text
- `feeCents`: integer from `0` to `1000000` cents (a course fee)
- `capacity`: integer from `1` to `1000` (the number of places of a course)
- `enrolledCount`: count

A record is identified by `courseId`.
