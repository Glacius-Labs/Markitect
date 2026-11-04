---
kind: Operation
---
# List shifts

Returns the scheduled shifts of all users that overlap the date, that is the interval from `00:00` of the date to `00:00` of the next date. They are ordered by `start` ascending, then in creation order.

- **Wire name:** `listShifts`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `date`: date
- **Result:**
  - `date`: date
  - `shifts`: a list of [[staffing/shift]] records
- **Permissions:** staff all, admin all.
