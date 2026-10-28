---
kind: std/Concept
---
# User

A user of the program: the initial admin, staff and admins registered by an admin, and members registered by staff or admins.

**Fields**, in the formats of [[values]]:

- `userId`: identifier
- `name`: text
- `role`: one of `member`, `staff` or `admin`; see authorization

A record is identified by `userId`.
