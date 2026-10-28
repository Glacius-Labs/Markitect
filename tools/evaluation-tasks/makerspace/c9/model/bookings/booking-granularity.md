---
kind: std/Rule
internal: true
---
# Booking granularity

**Booking granularity is 15 minutes.** A reservation's start and end must each fall on a minute that is a multiple of 15 within the hour (`:00`, `:15`, `:30` or `:45`). A reservation lasts at least one granularity step (15 minutes) and at most 240 minutes. Check-in and check-out times are not tied to the granularity.
