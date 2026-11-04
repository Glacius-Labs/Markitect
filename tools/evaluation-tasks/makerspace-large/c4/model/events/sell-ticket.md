---
kind: Operation
---
# Sell ticket

Sells a ticket to the member: a ticket is created with status `valid` and the next ticket identifier, and the event's `sold` increases by one. An event [[billing/charge]] is created for the member: `kind` `event`, `ref` the ticket identifier, `quantity` 1, `unitPriceCents` the event's `priceCents`, `discountPercent` 0, `at` the request time. An event with price 0 creates a charge of 0 cents. The `event` result is the event after the change.

Rules of other modules that apply here: [[billing/charges-and-invoices]] and [[members/membership]].

- **Wire name:** `sellTicket`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `eventId`: identifier, naming an existing [[events/event]]
  - `memberId`: identifier, naming an existing [[members/member]]
- **Refusals** at check 7, in this order:
  1. `member_suspended` when the member is not in good standing ([[members/members-interface]]).
  2. `event_closed` when the event is cancelled or its `start` is at or before the request time.
  3. `already_ticketed` when the member holds a valid ticket for the event.
  4. `event_sold_out` when `sold` equals `capacity`.
- **Result:**
  - `ticket`: a [[events/ticket]] record
  - `event`: an [[events/event]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** the new `ticketId`.
