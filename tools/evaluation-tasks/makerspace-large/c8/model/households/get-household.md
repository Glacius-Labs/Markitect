---
kind: Operation
---
# Get household

Returns the household.

- **Wire name:** `getHousehold`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `householdId`: identifier, naming an existing [[households/household]]
- **Result:**
  - `household`: a [[households/household]] record
- **Permissions:** staff all, admin all.
