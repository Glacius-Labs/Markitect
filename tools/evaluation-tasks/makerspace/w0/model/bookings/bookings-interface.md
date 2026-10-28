---
kind: Interface
---
# Bookings interface

What other modules may use of bookings: its operations and the reservation record. Other modules also ask whether an active reservation of a machine overlaps an interval, and for the completed sessions of a machine.

- **Operations:** [[bookings/create-reservation]], [[bookings/cancel-reservation]], [[bookings/check-in]], [[bookings/check-out]], [[bookings/list-member-reservations]], [[bookings/get-machine-schedule]].
- **Records:** [[bookings/reservation]], [[bookings/opening-hours-of-day]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `check_in_too_early` | The reservation has not started yet. |
| `invalid_slot` | A reservation does not follow the booking granularity or its duration limits. |
| `machine_booked` | Another active reservation of the machine overlaps the reservation. |
| `machine_in_maintenance` | A maintenance window of the machine overlaps the reservation. |
| `member_double_booked` | Another active reservation of the member overlaps the reservation. |
| `outside_opening_hours` | A reservation is not inside the opening hours of its day. |
