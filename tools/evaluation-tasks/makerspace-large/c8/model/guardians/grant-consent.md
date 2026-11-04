---
kind: Operation
---
# Grant consent

Adds the family to the member's `consentFamilyIds`, which stays ascending. Granting a consent that is held already succeeds without a change. The member does not have to be a minor any more. Ownership is judged on `guardianId` alone: the guardian is the actor.

- **Wire name:** `grantConsent`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `guardianId`: identifier, naming an existing [[members/member]]
  - `memberId`: identifier, naming an existing [[members/member]]
  - `familyId`: identifier, naming an existing [[equipment/family]]
- **Refusals** at check 7, in this order:
  1. `not_guardian` when the member has no guardianship, or its `guardianId` differs from the argument.
- **Result:**
  - `guardianship`: a [[guardians/guardianship]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `memberId/familyId`.
