---
kind: Operation
---
# Register user

Registers a staff member, an admin or a steward: creates the user.

- **Wire name:** `registerUser`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `userId`: identifier
  - `name`: text
  - `role`: one of `staff`, `admin` or `steward`
- **Refusals** at check 7, in this order:
  1. `already_exists` when any user, including a member, has the identifier `userId`.
- **Result:**
  - `user`: a [[members/user]] record
- **Permissions:** admin all.
- **Audit target:** `userId`.
