---
kind: std/Rule
internal: true
---
# Reservation rules

A new reservation is accepted only when all of the following hold. These rules connect members, training, equipment and time, and the records of the workshop floor:

1. its slot follows the [[bookings/booking-granularity|booking granularity]] and duration limits and starts at or after the request time;
2. the slot lies inside the [[bookings/opening-hours|opening hours]] of its start date;
3. the member is in good standing ([[members/membership]]);
4. a member who is a minor on the date of `start` holds consent for the machine's family ([[guardians/guardian-consent]]);
5. the member holds a certification for the machine's family that is valid on the reservation date, which is the date of `start` ([[training/certifications]]);
6. the member's latest signature for the machine's family is of the current waiver version ([[waivers/waiver-gate]]);
7. the machine has no active lockout at the request time ([[lockouts/lockout-gate]]);
8. the date of `start` is before the machine's next inspection date, when it has an inspection plan ([[inspections/inspection-gate]]);
9. no scheduled maintenance window of the machine overlaps the slot ([[equipment/maintenance]]);
10. the slot is covered by scheduled shifts, when the machine's family is supervised ([[staffing/supervision]]);
11. no other active reservation of the machine overlaps the slot;
12. no other active reservation of the member, on any machine, overlaps the slot.

Rules 4, 6, 7, 8 and 10 bind only where the record they name exists, as their own pages state.
