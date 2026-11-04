---
kind: std/Rule
internal: true
---
# Ticket sales

An [[events/event]] sells up to `capacity` [[events/ticket|tickets]] at `priceCents` each. Its `sold` is the number of its valid tickets. A sale raises it by one and the cancellation of a ticket lowers it by one, so a cancelled ticket gives its place back. The event is **sold out** when `sold` equals `capacity`. A member holds at most one valid ticket per event and may buy another one after cancelling.

An event is **closed** when it is cancelled or its `start` is at or before the request time. Tickets are sold only for an event that is not closed. A cancelled event stays cancelled; its `sold` is 0.

A ticket's charge is created when it is sold and is never changed. When a ticket is cancelled, by [[events/cancel-ticket]] or together with its event by [[events/cancel-event]], the member's credit balance increases by the event's `priceCents`, through the add-credit capability of [[billing/billing-interface]], and the posting that [[ledger/postings]] gives for the event is made. The credit does not depend on whether the charge is invoiced or paid, and a price of 0 gives no credit and no posting.

An event keeps its room: creating it makes a hold of the room for `[start, end)` with origin `event` and purpose `event ` followed by the event identifier, and cancelling it releases that hold ([[rooms/room-holds]]). The capacity of an event is judged against the capacity of its room once, when the event is created.
