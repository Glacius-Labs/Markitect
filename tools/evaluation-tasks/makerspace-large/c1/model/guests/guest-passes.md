---
kind: std/Rule
internal: true
---
# Guest passes

A member's tier decides the [[guests/guest-allowance]]. The passes of a member that **count** for a month are the member's guest passes with status `issued` whose `date` lies in that month; a revoked pass does not count. The month of a pass is the month of its `date`, not of the request time.

A new pass is **free** when the member's tier has an allowance and fewer than `perMonth` passes count for the month of its date. Otherwise, if the tier has an allowance, it is **charged**. With `perMonth` 0 every pass is charged. The tier is the member's tier at the request time. A pass that is charged has `charged` true and one [[billing/charge]]; its `feeCents` may be 0, and the charge of 0 cents is still created. Revoking a pass takes no charge back and does not change its `charged` field.

Example: the tier has `perMonth` 2 and `feeCents` 500. The member's passes for `2026-06-10`, `2026-06-11` and `2026-06-12` are free, free and charged 500. After the first is revoked, 2 passes count for June, so a pass for `2026-06-20` is charged. A pass for `2026-07-01` is free, because no pass counts for July.
