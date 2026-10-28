---
kind: Interface
---
# Members interface

What other modules may use of members: its operations, the user and member records, tier and standing, and the refusal `member_suspended`. Other modules also ask whether a member exists, whether the member is in good standing, and the member's tier with its monthly fee and usage discount.

- **Operations:** [[members/register-user]], [[members/register-member]], [[members/get-member]], [[members/change-tier]], [[members/set-member-standing]].
- **Records:** [[members/user]], [[members/member]], [[members/tier]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `member_suspended` | The member is not in good standing. |
