---
kind: std/Rule
---
# Waiver gate

A family has a waiver once one is published. A family without a waiver requires no signature. A member **has signed** a family's waiver when the member's latest [[waivers/waiver-signature]] for the family has the family's current version.

**The gate.** A member who has not signed the waiver of the machine's family, because there is no signature or the latest is of an older version, is refused `waiver_missing`. It applies at `createReservation` and `checkIn`, at the place each lists. Standing, certification and consent do not change it.

Publishing a new version leaves existing reservations in place. Their check-in is refused with `waiver_missing` until the member signs the new version.
