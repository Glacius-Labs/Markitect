---
kind: std/Rule
---
# Reservations

A [[bookings/reservation]] books one machine for one member for an interval `[start, end)` on one day. Its stored status moves from `reserved` to `checkedIn` to `completed`, or from `reserved` to `cancelled`. Its reported status is the stored status, except that a stored `reserved` reservation whose `end` is at or before the request time is reported as `missed`. A missed reservation can no longer be checked in or cancelled and causes no charge.

A reservation is **active** when its reported status is `reserved` or `checkedIn`. Only active reservations block a machine or a member. Conflicts are judged on the reserved interval `[start, end)`, not on the actual check-in and check-out times.
