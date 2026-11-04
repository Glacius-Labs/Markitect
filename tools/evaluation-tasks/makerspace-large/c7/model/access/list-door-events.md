---
kind: Operation
---
# List door events

Returns the door events whose `at` lies on the date, in creation order.

- **Wire name:** `listDoorEvents`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `date`: date
- **Result:**
  - `date`: date
  - `events`: a list of [[access/door-event]] records
- **Permissions:** staff all, admin all.
