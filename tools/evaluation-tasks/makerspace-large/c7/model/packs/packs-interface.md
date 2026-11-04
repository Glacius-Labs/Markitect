---
kind: Interface
---
# Packs interface

What other modules may use of packs: its operations and the pack product and pack records. Bookings also asks packs to draw minutes at check-out.

- **Operations:** [[packs/add-pack-product]], [[packs/sell-pack]], [[packs/list-member-packs]].
- **Uses:** [[members/members-interface]], [[billing/billing-interface]].
- **Records:** [[packs/pack-product]], [[packs/pack]].
- **Capabilities** that are not operations:
  - **Draw minutes.** The caller names a member and a number of metered minutes. Packs draws from the member's packs as [[packs/pack-draw|the pack draw rule]] states, lowers their `minutesLeft` and returns the number of minutes drawn, which is never more than the minutes asked for.
