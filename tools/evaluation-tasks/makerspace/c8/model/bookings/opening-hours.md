---
kind: std/Rule
---
# Opening hours

**Opening hours** are the same every week:

| Day | Opens | Closes | Open minutes |
| --- | --- | --- | --- |
| Monday to Thursday | 08:00 | 22:00 | 840 |
| Friday | 08:00 | 20:00 | 720 |
| Saturday | 09:00 | 17:00 | 480 |
| Sunday | 12:00 | 16:00 | 240 |

The opening period of a day is the interval `[opens, closes)` on that date. A reservation is inside opening hours when its whole interval lies in the opening period of its start date: its start is at or after the opening time and its end is at or before the closing time of that same date. A reservation ending exactly at closing time is inside. Every day has an opening period. The hours apply to every date, including dates before this change.
