---
kind: std/Concept
---
# Class

One session of a course, taught by one instructor in one room for an interval, generated identifier `cls-N`.

**Fields**, in the formats of [[values]]:

- `classId`: identifier
- `courseId`: identifier, naming an existing [[training/course]]
- `instructorId`: identifier, naming an existing [[members/user]]
- `roomId`: identifier, naming an existing [[rooms/room]]
- `start`: timestamp
- `end`: timestamp
- `status`: `scheduled` or `cancelled`

A record is identified by `classId`.
