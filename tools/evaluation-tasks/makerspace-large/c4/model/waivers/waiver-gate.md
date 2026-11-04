---
kind: std/Rule
---
# Waiver gate

A family has a waiver once one is published. A family without a waiver requires no signature. A [[waivers/waiver-signature]] has a **lapse date**, the date 365 days after the date of its `signedAt`. It counts up to the day before its lapse date and not on the lapse date itself, so a signature with `signedAt` `2026-03-02T09:00Z` has the lapse date `2027-03-02`, counts at `2027-03-01T23:59Z` and does not count at `2027-03-02T00:00Z`. A member **has signed** a family's waiver when the member's latest signature for the family has the family's current version and has not lapsed on the date of the request time. The date of a slot plays no part in this.

**The gate.** A member who has not signed the waiver of the machine's family, because there is no signature, the latest is of an older version or the latest has lapsed, is refused `waiver_missing`. It applies at `createReservation` and `checkIn`, at the place each lists, and it is judged at the request time of the request. Standing, certification and consent do not change it.

Publishing a new version leaves existing reservations in place, and so does the lapse of a signature. Their check-in is refused with `waiver_missing` until the member signs the current version again.
