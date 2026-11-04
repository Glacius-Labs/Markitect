---
kind: std/Rule
---
# Lockout gate

A machine is **locked out** at a request time when it has at least one active lockout ([[lockouts/lockouts]]). `createReservation` and `checkIn` refuse `machine_locked_out` when the machine is locked out at the request time.

The gate looks at the lockouts as they stand when the request is answered. It does not look at the date of the slot: a lockout that is active now refuses a reservation for any date, and a lockout that is released now does not refuse one. A reservation made earlier stays in place, and its check-in is refused while the machine is locked out.

A session that is already checked in is not interrupted. `checkOut` is never refused because of a lockout, and the metering of the session goes on.

Example: staff apply a lockout to `laser-1` at `2026-03-02T09:00Z`. At `2026-03-02T10:00Z` a reservation of `laser-1` for `2026-04-01` is refused with `machine_locked_out`, although the lockout says nothing about April. Once the lockout is released, the same request is accepted.
