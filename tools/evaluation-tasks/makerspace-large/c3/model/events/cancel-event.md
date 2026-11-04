---
kind: Operation
---
# Cancel event

The event gets status `cancelled` and its room hold is released. Every valid ticket of the event, in creation order, gets status `cancelled`, with the credit and the posting that [[events/ticket-sales]] states for each. `sold` becomes 0. The 1440-minute limit of [[events/cancel-ticket]] does not apply. `tickets` are the tickets cancelled by this request, in creation order; the list is empty when the event had no valid ticket.

- **Wire name:** `cancelEvent`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `eventId`: identifier, naming an existing [[events/event]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the event is already cancelled or its `start` is at or before the request time.
- **Result:**
  - `event`: an [[events/event]] record
  - `tickets`: a list of [[events/ticket]] records
- **Permissions:** staff all, admin all.
- **Audit target:** `eventId`.
