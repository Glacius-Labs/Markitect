---
kind: Operation
---
# Sign waiver

Records a signature of the family's current waiver: `version` is the current version and `signedAt` the request time. Earlier signatures stay. Standing is not checked. `already_signed` is judged as [[waivers/waiver-gate]] defines having signed.

Rules of other modules that apply here: [[guardians/guardian-consent]].

- **Wire name:** `signWaiver`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
  - `familyId`: identifier, naming an existing [[equipment/family]]
- **Refusals** at check 7, in this order:
  1. `no_waiver` when the family has no waiver.
  2. `already_signed` when the member has signed the family's waiver.
  3. `guardian_required` when the member is a minor on the date of the request time and holds no consent for the family.
- **Result:**
  - `signature`: a [[waivers/waiver-signature]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `memberId/familyId`.
