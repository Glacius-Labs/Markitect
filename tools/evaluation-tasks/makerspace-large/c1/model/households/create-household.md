---
kind: Operation
---
# Create household

Creates the household. Its `memberIds` hold only the payer.

- **Wire name:** `createHousehold`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `householdId`: identifier
  - `name`: text
  - `payerId`: identifier, naming an existing [[members/member]]
- **Refusals** at check 7, in this order:
  1. `already_exists` when a household with `householdId` exists.
  2. `already_in_household` when the payer belongs to a household.
- **Result:**
  - `household`: a [[households/household]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `householdId`.
