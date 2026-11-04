---
kind: Operation
---
# Compliance summary

Shows where the member stands. `standing` is the member's standing. `minor` is `true` when the member is a minor on the date of the request time ([[guardians/guardian-consent]]), otherwise `false`. `families` has one [[compliance/family-compliance]] row for every family that exists at the request time, in `familyId` ascending order, and is `[]` when there are no families.

- **Wire name:** `complianceSummary`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
- **Result:**
  - `memberId`: identifier
  - `standing`: `good`, `suspended` or `paused`
  - `minor`: flag
  - `families`: a list of [[compliance/family-compliance]] records
- **Permissions:** member own, staff all, admin all.
