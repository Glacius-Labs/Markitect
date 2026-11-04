---
kind: Operation
---
# Issue badge

Creates a badge for the member with the next badge identifier, status `active` and `inside` false. Standing is not checked.

- **Wire name:** `issueBadge`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
- **Refusals** at check 7, in this order:
  1. `badge_active` when the member has an active badge.
- **Result:**
  - `badge`: a [[access/badge]] record
- **Permissions:** staff all, admin all.
- **Audit target:** the new `badgeId`.
