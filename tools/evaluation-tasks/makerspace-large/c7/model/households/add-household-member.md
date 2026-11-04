---
kind: Operation
---
# Add household member

Adds the member to the household. `memberIds` stays ascending. The payer is not changed.

- **Wire name:** `addHouseholdMember`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `householdId`: identifier, naming an existing [[households/household]]
  - `memberId`: identifier, naming an existing [[members/member]]
- **Refusals** at check 7, in this order:
  1. `already_in_household` when the member belongs to a household, this one or another.
  2. `household_full` when the household has 6 members.
- **Result:**
  - `household`: a [[households/household]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `householdId/memberId`.
