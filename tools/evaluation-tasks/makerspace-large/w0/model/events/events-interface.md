---
kind: Interface
---
# Events interface

What other modules may use of events: its operations and the event and ticket records. No other module asks anything else of events.

- **Operations:** [[events/create-event]], [[events/sell-ticket]], [[events/cancel-ticket]], [[events/cancel-event]], [[events/list-event-tickets]].
- **Uses:** [[members/members-interface]], [[rooms/rooms-interface]], [[billing/billing-interface]], [[ledger/ledger-interface]].
- **Records:** [[events/event]], [[events/ticket]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_ticketed` | The member holds a valid ticket for the event. |
| `capacity_too_large` | The event's capacity is above the capacity of its room. |
| `event_closed` | The event is cancelled, or it starts at or before the request time. |
| `event_sold_out` | All places of the event are taken by valid tickets. |
| `too_late` | The event starts in less than 1440 minutes. |
