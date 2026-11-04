---
kind: Operation
---
# List class attendance

Returns the attendance marks of the class, one for each member who has a mark, in ascending `memberId` order. A class with no marks returns an empty list.

- **Wire name:** `listClassAttendance`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `classId`: identifier, naming an existing [[classes/class]]
- **Result:**
  - `classId`: identifier
  - `attendance`: a list of [[classes/attendance]] records
- **Permissions:** staff all, admin all.
