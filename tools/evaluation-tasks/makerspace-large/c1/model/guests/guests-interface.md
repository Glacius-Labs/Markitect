---
kind: Interface
---
# Guests interface

What other modules may use of guests: its operations, the guest allowance and guest pass records and the refusal codes below. Other modules also ask for a guest pass by its identifier, with its member, date and status.

- **Operations:** [[guests/set-guest-allowance]], [[guests/issue-guest-pass]], [[guests/revoke-guest-pass]], [[guests/list-guest-passes]].
- **Uses:** [[members/members-interface]], [[billing/billing-interface]].
- **Records:** [[guests/guest-allowance]], [[guests/guest-pass]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `date_in_past` | The date is before the date of the request time. |
| `guests_not_allowed` | The member's tier has no guest allowance. |
