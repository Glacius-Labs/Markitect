---
kind: Operation
---
# Get guardianship

Returns the member's guardianship.

- **Wire name:** `getGuardianship`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
- **Refusals** at check 7, in this order:
  1. `not_a_declared_minor` when the member has no guardianship.
- **Result:**
  - `guardianship`: a [[guardians/guardianship]] record
- **Permissions:** member own, staff all, admin all.
