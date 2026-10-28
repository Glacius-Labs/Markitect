---
kind: Operation
---
# Register member

Registers a member with a membership: creates a user with role `member`, the given tier, standing `good`, `registeredAt` equal to the request time and a credit balance of 0.

- **Wire name:** `registerMember`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier
  - `name`: text
  - `tier`: a [[members/tier]]: `basic`, `plus` or `pro`
- **Refusals** at check 7, in this order:
  1. `already_exists` when any user has the identifier `memberId`.
- **Result:**
  - `member`: a [[members/member]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `memberId`.
