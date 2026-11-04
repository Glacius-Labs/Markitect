---
kind: Operation
---
# Cancel ticket

The ticket gets status `cancelled` and the event's `sold` decreases by one. The member's credit balance increases by the event's `priceCents` and a posting is made, as [[events/ticket-sales]] states. The ticket's charge does not change. The `event` result is the event after the change.

A ticket can be cancelled until 1440 minutes before the event's start. Example: for an event that starts at `2026-03-10T18:00Z`, a request at `2026-03-09T18:00Z` is accepted and a request at `2026-03-09T18:01Z` is refused with `too_late`.

- **Wire name:** `cancelTicket`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `ticketId`: identifier, naming an existing [[events/ticket]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the ticket is already cancelled.
  2. `too_late` when the `start` of the ticket's event is less than 1440 minutes after the request time, which includes an event that has started.
- **Result:**
  - `ticket`: an [[events/ticket]] record
  - `event`: an [[events/event]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `ticketId`.
