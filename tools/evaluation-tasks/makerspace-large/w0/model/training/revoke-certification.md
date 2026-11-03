---
kind: Operation
---
# Revoke certification

Withdraws the member's certification, for example after a safety incident. The certification becomes revoked; its dates do not change. Existing reservations stay, but check-in requires a valid certification. An expired certification can be revoked.

- **Wire name:** `revokeCertification`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
  - `familyId`: identifier, naming an existing [[equipment/family]]
- **Refusals** at check 7, in this order:
  1. `not_certified` when the member has no certification for the family or it is already revoked.
- **Result:**
  - `certification`: a [[training/certification]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `memberId/familyId`.
