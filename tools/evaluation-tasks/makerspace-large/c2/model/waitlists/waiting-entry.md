---
kind: std/Concept
---
# Waiting entry

The place of one member on the waiting list of one course.

**Fields**, in the formats of [[values]]:

- `courseId`: identifier, naming an existing [[training/course]]
- `memberId`: identifier, naming an existing [[members/member]]
- `joinedAt`: timestamp

A record is identified by `courseId` and `memberId` together.
