---
kind: std/Rule
internal: true
---
# Reservation rules

A new reservation is accepted only when all of the following hold. These rules connect members, training, equipment and time:

1. its slot follows the [[bookings/booking-granularity|booking granularity]] and duration limits and starts at or after the request time;
2. the slot lies inside the [[bookings/opening-hours|opening hours]] of its start date;
3. the member is in good standing ([[members/membership]]);
4. the member is not blocked by a long-overdue invoice ([[billing/account-blocks|blocked members]]);
5. the member holds a certification for the machine's family that is not revoked ([[training/certifications]]);
6. no scheduled maintenance window of the machine overlaps the slot ([[equipment/maintenance]]);
7. no lockout of the machine overlaps the slot ([[bookings/lockouts]]);
8. no other active reservation of the machine overlaps the slot;
9. no other active reservation of the member, on any machine, overlaps the slot.
