---
kind: Operation
---
# Revoke scholarship

Revokes the award: its status becomes `revoked`. The fund's `spentCents` does not change, and discounts already given stay. A revoked award gives no further discounts, and the member can be awarded again.

- **Wire name:** `revokeScholarship`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `awardId`: identifier, naming an existing [[scholarships/award]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the award's status is `revoked`.
- **Result:**
  - `award`: a [[scholarships/award]] record
- **Permissions:** admin all.
- **Audit target:** `awardId`.
