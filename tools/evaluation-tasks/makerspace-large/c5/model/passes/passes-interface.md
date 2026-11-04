---
kind: Interface
---
# Passes interface

What other modules may use of passes: its operations and the pass product and pass records. Bookings also asks passes to draw minutes at check-out.

- **Operations:** [[passes/add-pass-product]], [[passes/sell-pass]], [[passes/list-member-passes]].
- **Uses:** [[members/members-interface]], [[billing/billing-interface]].
- **Records:** [[passes/pass-product]], [[passes/pass]].
- **Capabilities** that are not operations:
  - **Draw minutes.** The caller names a member and a number of metered minutes. Passes draws from the member's passes as [[passes/pass-draw|the pass draw rule]] states, lowers their `minutesLeft` and returns the number of minutes drawn, which is never more than the minutes asked for.
