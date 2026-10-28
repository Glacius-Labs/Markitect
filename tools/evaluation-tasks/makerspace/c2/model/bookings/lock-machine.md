---
kind: Operation
---
# Lock machine

Locks the machine of an open incident:

1. A lockout is created for the incident's machine, with the next lockout identifier.
2. Every reservation of that machine whose reported status is `reserved` and whose interval overlaps the lockout becomes `cancelled`. These are the reservations of the window that are not yet used: not checked in and not missed. This includes a reservation whose start has already passed but which is not yet checked in and not yet missed. Reservations that are checked in, completed, missed or already cancelled are not changed.
3. For each reservation cancelled this way, the reservation's member receives a lockout credit of 1000 cents: the member's credit balance increases by 1000, whatever the member's standing.

The request writes exactly one audit entry, for `lockMachine`; the cancellations and credits are part of that request and have no entries of their own. `cancelledReservations` are the reservations cancelled by this request, after the change, in creation order.

- **Wire name:** `lockMachine`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `incidentId`: identifier, naming an existing [[safety/incident]]
  - `start`: timestamp
  - `end`: timestamp
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the incident is resolved.
  2. `invalid_interval` when `end` is not after `start`.
  3. `starts_in_past` when `start` is before the request time.
- **Result:**
  - `lockout`: a [[bookings/lockout]] record
  - `cancelledReservations`: a list of [[bookings/reservation]] records
- **Permissions:** staff all, admin all.
- **Audit target:** the new `lockoutId`.
