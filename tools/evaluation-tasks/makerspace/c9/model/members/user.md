---
kind: std/Concept
---
# User

A user of the program: the initial admin, staff, admins and stewards registered by an admin, and members registered by staff or admins.

**Fields**, in the formats of [[values]]:

- `userId`: identifier
- `name`: text
- `role`: one of `member`, `staff`, `admin` or `steward`; see authorization

A record is identified by `userId`.
