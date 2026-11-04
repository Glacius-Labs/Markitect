---
kind: Operation
---
# Check eligibility

Tells whether the member could reserve the machine for a slot that starts on `date`, and what stops them. `blockers` holds the codes of the blockers that hold, in the order [[compliance/eligibility]] lists them, and is `[]` when none holds. `eligible` is `true` exactly when `blockers` is empty.

- **Wire name:** `checkEligibility`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
  - `machineId`: identifier, naming an existing [[equipment/machine]]
  - `date`: date
- **Result:**
  - `memberId`: identifier
  - `machineId`: identifier
  - `date`: date
  - `eligible`: flag
  - `blockers`: a list of refusal codes
- **Permissions:** member own, staff all, admin all.
