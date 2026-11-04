---
kind: std/Rule
---
# Membership

A membership has a [[members/tier]] and a [[members/member|standing]]. The fee of the tier is charged once for every invoiced month, without proration, except for a paused member. The usage discount applies only to usage charges, never to course or material charges.

A member is **in good standing** when the standing is `good`. A suspended or paused member cannot make reservations, check in, buy materials, enroll in courses, borrow tools or renew loans. Suspension and pausing do not cancel existing reservations or enrollments. A suspended or paused member can still cancel reservations, check out, return tools, see their records and pay invoices. Reinstating good standing removes these restrictions; all other rules still apply.

A suspended member still pays the monthly fee of the tier. The monthly fee of a **paused** member is 500 cents, whatever the tier. It is the standing at the request time of [[billing/generate-invoices]] that counts, whatever month is invoiced. A paused member is still a member and is invoiced every month like any other.

Example: a paused member on `pro` pays 500 cents, and 450 cents (`(500 × 90 + 50) div 100`) in a household of 3 or more ([[households/household-discount]]).
