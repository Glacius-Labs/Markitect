---
kind: Operation
---
# List waitlist

Returns the entries of the course's waiting list in join order. A course with no entries returns an empty list.

- **Wire name:** `listWaitlist`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `courseId`: identifier, naming an existing [[training/course]]
- **Result:**
  - `courseId`: identifier
  - `entries`: a list of [[waitlists/waiting-entry]] records
- **Permissions:** staff all, admin all.
