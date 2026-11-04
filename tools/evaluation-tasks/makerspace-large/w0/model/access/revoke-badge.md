---
kind: Operation
---
# Revoke badge

The badge gets status `revoked` and `inside` false. No door event is created. The member may be issued a new badge afterwards.

- **Wire name:** `revokeBadge`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `badgeId`: identifier, naming an existing [[access/badge]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the badge is already revoked.
- **Result:**
  - `badge`: a [[access/badge]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `badgeId`.
