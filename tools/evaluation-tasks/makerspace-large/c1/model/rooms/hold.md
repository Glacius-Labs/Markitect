---
kind: std/Concept
---
# Hold

A hold of a room for an interval, generated identifier `hld-N`. The hold keeps the room for `[start, end)` while its status is `held`.

**Fields**, in the formats of [[values]]:

- `holdId`: identifier
- `roomId`: identifier, naming an existing [[rooms/room]]
- `start`: timestamp
- `end`: timestamp
- `purpose`: text
- `status`: `held` or `released`

A record is identified by `holdId`.
