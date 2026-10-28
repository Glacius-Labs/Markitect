---
kind: std/Concept
---
# Reservation

A reservation of one machine for one member, generated identifier `res-N`. `checkedInAt` and `rateCentsPerMinute` are `null` until check-in; `checkedOutAt`, `meteredMinutes` and `amountCents` are `null` until check-out.

**Fields**, in the formats of [[values]]:

- `reservationId`: identifier
- `memberId`: identifier, naming an existing [[members/member]]
- `machineId`: identifier, naming an existing [[equipment/machine]]
- `start`: timestamp
- `end`: timestamp
- `status`: one of `reserved`, `missed`, `checkedIn`, `completed` or `cancelled`
- `checkedInAt`: timestamp, or `null`
- `checkedOutAt`: timestamp, or `null`
- `rateCentsPerMinute`: integer from `1` to `10000` cents (a machine's rate per minute), or `null`
- `meteredMinutes`: minute count, or `null`
- `amountCents`: cent amount, or `null`

A record is identified by `reservationId`.
