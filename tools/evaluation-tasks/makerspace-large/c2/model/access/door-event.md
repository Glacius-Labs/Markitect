---
kind: std/Concept
---
# Door event

One passage at the door, generated identifier `dor-N`. `guestPassId` is `null` unless the kind is `guest`.

**Fields**, in the formats of [[values]]:

- `eventId`: identifier
- `kind`: a [[access/door-event-kind]]: `enter`, `leave` or `guest`
- `badgeId`: identifier, naming an existing [[access/badge]]
- `guestPassId`: identifier, naming an existing [[guests/guest-pass]], or `null`
- `at`: timestamp

A record is identified by `eventId`.
