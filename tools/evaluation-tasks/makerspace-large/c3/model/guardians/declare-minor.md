---
kind: Operation
---
# Declare minor

Creates the member's guardianship with the given birth date and guardian and no consent. Standing is not checked.

Rules of other modules that apply here: [[guardians/guardian-consent]].

- **Wire name:** `declareMinor`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
  - `birthDate`: date
  - `guardianId`: identifier, naming an existing [[members/member]]
- **Refusals** at check 7, in this order:
  1. `already_exists` when the member has a guardianship.
  2. `not_a_minor` when the birth date makes the member an adult on the date of the request time.
  3. `invalid_guardian` when `guardianId` equals `memberId`, or the guardian is a minor on the date of the request time.
- **Result:**
  - `guardianship`: a [[guardians/guardianship]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `memberId`.
