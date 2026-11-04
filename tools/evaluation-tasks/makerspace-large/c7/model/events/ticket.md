---
kind: std/Concept
---
# Ticket

The ticket of one member for one event, generated identifier `tkt-N`.

**Fields**, in the formats of [[values]]:

- `ticketId`: identifier
- `eventId`: identifier, naming an existing [[events/event]]
- `memberId`: identifier, naming an existing [[members/member]]
- `status`: `valid` or `cancelled`

A record is identified by `ticketId`.
