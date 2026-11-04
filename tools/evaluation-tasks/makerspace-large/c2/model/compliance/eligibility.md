---
kind: std/Rule
internal: true
---
# Eligibility

A member is **eligible** for a machine on a date when none of the following **blockers** holds. Each blocker is judged as `createReservation` judges its refusal of the same name for a slot that starts on that date, at the request time. The blockers are listed in the order of that operation's refusals:

1. `member_suspended`: the member is not in good standing ([[members/members-interface]]).
2. `guardian_consent_missing`: the member is a minor on the date and holds no consent for the machine's family ([[guardians/guardian-consent]]).
3. `not_certified`: the member has no certification for the machine's family valid on the date ([[training/training-interface]]).
4. `waiver_missing`: the [[waivers/waiver-gate]] refuses the member for the machine's family at the request time, whatever the date.
5. `machine_locked_out`: the machine has an active lockout at the request time, whatever the date ([[lockouts/lockout-gate]]).
6. `inspection_due`: the machine has an inspection plan and the date is on or after its `nextDueOn` ([[inspections/inspection-gate]]).

The rules that depend on the slot itself are not judged: granularity, opening hours, a start in the past, maintenance, supervision and conflicts with other reservations. The check-in rules `ppe_missing` and `not_on_premises` are not judged either. The date may lie in the past.

Example: a member in good standing, certified from `2026-01-10` for 365 days, asks about a machine on `2026-03-31`. The machine has a plan with `nextDueOn` `2026-03-31` and no lockout. The only blocker is `inspection_due`; for `2026-03-30` the member is eligible.
