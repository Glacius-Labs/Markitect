---
kind: std/Rule
---
# No-show fee

A member who books a machine and does not come blocks the slot for everyone else, and pays half the price of the time blocked. The fee is charged only by a run that staff start by hand ([[bookings/charge-no-shows]]); a missed reservation causes no charge by itself.

A run takes every [[bookings/reservation]] whose reported status is `missed` at the request time ([[bookings/reservations]]), in ascending reservation number: the number `N` of `res-N`, compared as a number. A reservation whose `end` is the request time is missed and is taken. For each one that no earlier run has charged and whose machine was not locked out during the slot, the run creates a [[billing/charge]] for the reservation's member: `kind` `noShow`, `ref` the reservation identifier, `quantity` the reserved minutes (`end − start`), `unitPriceCents` the machine's current rate at the request time, `discountPercent` 50, `at` the request time, and the amount by [[money]]. The member's tier discount does not apply; the 50 is the whole discount.

- The machine **was locked out during the slot** when a lockout of the machine, of any source, overlaps `[start, end)`, as [[lockouts/lockouts-interface]] states. Such a reservation is not charged, by this run or by any later one.
- A reservation that a run has charged is not charged again by a later run. A reservation that is checked in, completed, cancelled or still reserved is not missed and is not charged.
- The member's standing is not checked.
- The reservation does not change: it stays `missed`, and its `amountCents` stays `null`. No minutes of a pass and no stock are drawn, because both happen only at `checkOut`.

The charge is unbilled until the invoice of the month of its `at` takes it ([[billing/charges-and-invoices]]), which is the month of the run, whatever month the slot was in.

Example: at a rate of 45 cents per minute, a missed reservation of 60 minutes gives `(60 × 45 × 50 + 50) div 100 = 1350` cents, and one of 15 minutes gives `(15 × 45 × 50 + 50) div 100 = 338` cents.
