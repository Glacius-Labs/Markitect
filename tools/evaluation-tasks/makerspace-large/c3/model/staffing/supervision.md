---
kind: std/Rule
---
# Supervision

A family is **supervised** when its [[staffing/supervision-setting]] has `required` true. Only scheduled [[staffing/shift|shifts]] count; cancelled shifts count for nothing.

**Cover.** An interval is **covered** when every minute of it lies in at least one scheduled shift. The shifts may belong to several users and may touch or overlap, so cover can be pieced together. Example: the shifts `09:00`–`12:00` and `12:00`–`15:00` cover `11:00`–`13:00`; the shifts `09:00`–`12:00` and `12:30`–`15:00` do not.

**At reservation.** A reservation of a machine of a supervised family is refused `supervisor_required` when its slot is not covered. This applies at `createReservation`, at the place that operation lists. Turning supervision on does not touch existing reservations, and `checkIn` does not judge it.

**At cancellation.** [[staffing/cancel-shift]] is refused `shift_needed` when an active reservation of a machine of a supervised family overlaps the shift, is covered now and would not be covered if the shift were cancelled. A reservation is active as reservations states, judged on its reserved interval at the request time. A reservation that is not covered now does not stop a cancellation.

Example: the family is supervised, the shifts are `09:00`–`13:00` and `12:00`–`18:00` of one day, and an active reservation runs `12:00`–`13:30`. Cancelling the first shift is allowed, because the second still covers the reservation. Cancelling the second is refused, because `13:00`–`13:30` would be left uncovered.
