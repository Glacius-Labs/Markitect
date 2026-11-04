---
kind: Operation
---
# Remove household member

Removes the member from the household. The household stays, even when only the payer is left.

- **Wire name:** `removeHouseholdMember`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `householdId`: identifier, naming an existing [[households/household]]
  - `memberId`: identifier, naming an existing [[members/member]]
- **Refusals** at check 7, in this order:
  1. `not_in_household` when the member does not belong to this household.
  2. `payer_cannot_leave` when the member is the household's payer.
- **Result:**
  - `household`: a [[households/household]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `householdId/memberId`.
