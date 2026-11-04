---
kind: Interface
---
# Access interface

What other modules may use of access: its operations, the badge and door event records and the refusal codes below. Other modules also ask whether a member has an active badge that is not inside. The rule is [[access/premises-gate]]. Access in turn needs the opening period of a date from bookings, the account of a member from billing and the guest pass records of guests. Bookings, which access uses, declares its need for that answer, and the protocol module supplies it from access.

- **Operations:** [[access/issue-badge]], [[access/revoke-badge]], [[access/enter-door]], [[access/leave-door]], [[access/admit-guest]], [[access/list-door-events]].
- **Uses:** [[members/members-interface]], [[billing/billing-interface]], [[guests/guests-interface]], [[bookings/bookings-interface]].
- **Records:** [[access/badge]], [[access/door-event]], [[access/door-event-kind]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `account_overdue` | The member has an overdue balance above 0. |
| `already_inside` | The badge is already inside. |
| `badge_active` | The member has an active badge. |
| `guest_already_admitted` | The guest pass has been used to admit its guest already. |
| `not_inside` | The badge is not inside. |
| `not_on_premises` | The member has an active badge that is not inside. |
| `pass_not_today` | The guest pass is for another date than the date of the request time. |
| `pass_of_another_member` | The guest pass was issued to another member than the badge's. |
| `sponsor_not_inside` | The badge of the member who sponsors the guest is not inside. |
