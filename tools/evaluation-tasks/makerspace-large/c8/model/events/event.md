---
kind: std/Concept
---
# Event

A one-off event in a room with a number of tickets and a price, such as an open day.

**Fields**, in the formats of [[values]]:

- `eventId`: identifier
- `title`: text
- `roomId`: identifier, naming an existing [[rooms/room]]
- `start`: timestamp
- `end`: timestamp
- `capacity`: integer from `1` to `1000` (the number of tickets of an event)
- `priceCents`: integer from `0` to `1000000` cents (the price of one ticket)
- `status`: `open` or `cancelled`
- `sold`: count, the number of valid tickets of the event ([[events/ticket-sales]])

A record is identified by `eventId`.
