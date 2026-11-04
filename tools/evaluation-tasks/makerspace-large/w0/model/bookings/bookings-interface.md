---
kind: Interface
---
# Bookings interface

What other modules may use of bookings: its operations and the reservation record. Other modules also ask whether an active reservation of a machine overlaps an interval, and for the completed sessions of a machine. Bookings in turn needs to know whether a member has an active badge that is not inside; it declares that need, and the protocol module supplies the answer from access.

- **Operations:** [[bookings/create-reservation]], [[bookings/cancel-reservation]], [[bookings/check-in]], [[bookings/check-out]], [[bookings/list-member-reservations]], [[bookings/get-machine-schedule]].
- **Uses:** [[members/members-interface]], [[training/training-interface]], [[equipment/equipment-interface]], [[billing/billing-interface]], [[guardians/guardians-interface]], [[waivers/waivers-interface]], [[ppe/ppe-interface]], [[lockouts/lockouts-interface]], [[inspections/inspections-interface]], [[staffing/staffing-interface]], [[passes/passes-interface]], [[consumables/consumables-interface]].
- **Records:** [[bookings/reservation]], [[bookings/opening-hours-of-day]].

Capabilities other modules ask of bookings. None of them writes an audit entry of its own; the operation that uses it is audited once. "Active" and the reported status are as [[bookings/reservations]] states them, at the request time.

- Whether an active reservation of a machine overlaps an interval `[start, end)`. Equipment asks this.
- The completed sessions of a machine, each with its `checkedInAt` and `checkedOutAt`. Reporting asks this.
- The active reservations of all machines of a family, each with its machine, `start` and `end`. Staffing asks this.
- The reservations of a machine that are still to come, which are those whose reported status is `reserved` and whose `start` is after the request time, in ascending `start` order, then creation order. Escalations asks this.
- Cancelling a reservation whose reported status is `reserved` as part of another operation. The reservation gets status `cancelled` exactly as [[bookings/cancel-reservation]] states, with no fee and no credit. Escalations uses this.
- A reservation by its identifier, as the [[bookings/reservation]] record with its reported status. Projects asks this.
- The opening hours of a date, as an [[bookings/opening-hours-of-day]] record for that date's day of the week, or none on a Sunday ([[bookings/opening-hours]]). Access asks this to judge whether a time lies in the opening period of its date.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `check_in_too_early` | The reservation has not started yet. |
| `invalid_slot` | A reservation does not follow the booking granularity or its duration limits. |
| `machine_booked` | Another active reservation of the machine overlaps the reservation. |
| `machine_in_maintenance` | A maintenance window of the machine overlaps the reservation. |
| `member_double_booked` | Another active reservation of the member overlaps the reservation. |
| `outside_opening_hours` | A reservation is not inside the opening hours of its day. |

[[bookings/create-reservation]] and [[bookings/check-in]] also return codes that other modules add: `guardian_consent_missing`, `waiver_missing`, `machine_locked_out`, `inspection_due`, `supervisor_required`, `ppe_missing` and `not_on_premises`. Each is explained in the interface of the module that adds it.
