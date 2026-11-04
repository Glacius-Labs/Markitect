---
kind: Operation
---
# List event tickets

Returns all tickets of the event in every status, in creation order. An event with no tickets returns an empty list.

- **Wire name:** `listEventTickets`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `eventId`: identifier, naming an existing [[events/event]]
- **Result:**
  - `eventId`: identifier
  - `tickets`: a list of [[events/ticket]] records
- **Permissions:** staff all, admin all.
