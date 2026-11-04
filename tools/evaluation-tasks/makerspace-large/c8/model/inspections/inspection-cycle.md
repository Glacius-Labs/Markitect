---
kind: std/Rule
internal: true
---
# Inspection cycle

A machine has at most one [[inspections/inspection-plan]]. A machine without a plan is never inspected and never due.

`nextDueOn` is the date `intervalDays` days after `lastPassedOn` ([[time]]). It is computed again whenever `intervalDays` or `lastPassedOn` changes. It can lie after 2099 and then has a four-digit year ([[computed-times]]).

A passed inspection moves `lastPassedOn` to the date of the request time. A failed inspection leaves `lastPassedOn` and `nextDueOn` as they are and sets `failed`. The flag `failed` only records the latest outcome; it blocks nothing by itself. What blocks the machine after a failed inspection is the lockout of source `inspection` that the failure applies ([[lockouts/lockouts]]). Inspections keep at most one active lockout of source `inspection` on a machine: a second failure while one is active applies none, and a pass releases every one.

Example: a plan with `intervalDays` 30 is set on `2026-03-01`. `lastPassedOn` is `2026-03-01` and `nextDueOn` is `2026-03-31`. A passed inspection on `2026-03-31` moves `lastPassedOn` to `2026-03-31` and `nextDueOn` to `2026-04-30`.
