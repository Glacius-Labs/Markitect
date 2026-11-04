---
kind: Interface
---
# Households interface

What other modules may use of households: its operations, the household record and the refusal codes below. Other modules also ask which household a member belongs to, if any, and for that household its payer, its members in ascending order and their number. The rule that binds billing is [[households/household-discount]].

- **Operations:** [[households/create-household]], [[households/add-household-member]], [[households/remove-household-member]], [[households/get-household]].
- **Uses:** [[members/members-interface]].
- **Records:** [[households/household]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_in_household` | The member already belongs to a household. |
| `household_full` | The household already has 6 members. |
| `not_in_household` | The member does not belong to the household. |
| `payer_cannot_leave` | The member is the household's payer. |
