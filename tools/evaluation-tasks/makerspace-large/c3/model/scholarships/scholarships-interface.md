---
kind: Interface
---
# Scholarships interface

What other modules may use of scholarships: its operations, the fund and award records and the refusal `already_awarded`. Training also asks for a member's active award with the remaining budget of its fund, and adds to a fund's spent amount.

- **Operations:** [[scholarships/add-fund]], [[scholarships/award-scholarship]], [[scholarships/revoke-scholarship]], [[scholarships/list-awards]].
- **Uses:** [[members/members-interface]].
- **Records:** [[scholarships/fund]], [[scholarships/award]].
- **Capabilities** that are not operations, as [[scholarships/course-discount|the course discount rule]] states them:
  - **The active award.** The caller names a member. The answer is the member's active award with its `percent` and the remaining budget of its fund (`budgetCents` less `spentCents`), or none.
  - **Spend.** The caller names a fund and an amount in cents. The fund's `spentCents` increases by the amount.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_awarded` | The member already has an active award. |
