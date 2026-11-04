---
kind: Operation
---
# Withdraw consent

Removes the family from the member's `consentFamilyIds`. Existing reservations stay. Ownership is judged on `guardianId` alone: the guardian is the actor.

Rules of other modules that apply here: [[guardians/guardian-consent]].

- **Wire name:** `withdrawConsent`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `guardianId`: identifier, naming an existing [[members/member]]
  - `memberId`: identifier, naming an existing [[members/member]]
  - `familyId`: identifier, naming an existing [[equipment/family]]
- **Refusals** at check 7, in this order:
  1. `not_guardian` when the member has no guardianship, or its `guardianId` differs from the argument.
  2. `no_consent` when the family is not in `consentFamilyIds`.
- **Result:**
  - `guardianship`: a [[guardians/guardianship]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `memberId/familyId`.
