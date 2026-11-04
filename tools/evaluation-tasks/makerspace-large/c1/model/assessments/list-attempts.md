---
kind: Operation
---
# List attempts

Returns all attempts of the member for the course, from every enrollment period, in the order they were recorded. A member with no attempts gets an empty list.

- **Wire name:** `listAttempts`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `courseId`: identifier, naming an existing [[training/course]]
  - `memberId`: identifier, naming an existing [[members/member]]
- **Result:**
  - `courseId`: identifier
  - `memberId`: identifier
  - `attempts`: a list of [[assessments/attempt]] records
- **Permissions:** member own, staff all, admin all.
