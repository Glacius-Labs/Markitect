---
kind: Operation
---
# Get room schedule

Shows one room's day: `holds` are the room's held holds that overlap the interval from `00:00` of that date to `00:00` of the next date, in every origin. Released holds are not shown. A hold from `2026-03-03T23:00Z` to `2026-03-04T01:00Z` is shown for both `2026-03-03` and `2026-03-04`; a hold that ends at `2026-03-04T00:00Z` is not shown for `2026-03-04`.

- **Wire name:** `getRoomSchedule`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `roomId`: identifier, naming an existing [[rooms/room]]
  - `date`: date
- **Result:**
  - `roomId`: identifier
  - `date`: date
  - `holds`: a list of [[rooms/hold]] records
- **Permissions:** member all, staff all, admin all.
