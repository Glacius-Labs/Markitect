---
kind: std/Rule
---
# Inspection gate

`createReservation` refuses `inspection_due` when the machine has an [[inspections/inspection-plan]] and the date of the reservation's `start` is on or after the plan's `nextDueOn`. A machine without a plan is never refused. The plan's `failed` flag plays no part here.

Only the date of `start` counts. The request time, the date of `end` and the time of day do not. A reservation made before the machine fell due stays in place, and `checkIn` is not gated on the date, so it is not refused for a due inspection. After a passed inspection moves `nextDueOn`, reservations whose start date is before the new `nextDueOn` are accepted again.

Example: a plan has `nextDueOn` `2026-03-31`. A reservation starting on `2026-03-30` is accepted by this gate; one starting on `2026-03-31` or later is refused with `inspection_due`.
