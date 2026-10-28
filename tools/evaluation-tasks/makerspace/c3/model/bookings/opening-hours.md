---
kind: std/Rule
---
# Opening hours

**Opening hours** are the same every week:

| Day | Opens | Closes | Open minutes |
| --- | --- | --- | --- |
| Monday to Friday | 09:00 | 21:00 | 720 |
| Saturday | 10:00 | 18:00 | 480 |
| Sunday | closed | closed | 0 |

The opening period of a day is the interval `[opens, closes)` on that date. A reservation is inside opening hours when its whole interval lies in the opening period of its start date: its start is at or after the opening time and its end is at or before the closing time of that same date. A reservation ending exactly at closing time is inside. Sunday has no opening period.
