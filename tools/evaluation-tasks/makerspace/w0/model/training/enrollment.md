---
kind: std/Concept
---
# Enrollment

The enrollment of one member in one course. `completedAt` is `null` while enrolled.

**Fields**, in the formats of [[values]]:

- `courseId`: identifier, naming an existing [[training/course]]
- `memberId`: identifier, naming an existing [[members/member]]
- `status`: `enrolled` or `completed`
- `enrolledAt`: timestamp
- `completedAt`: timestamp, or `null`
