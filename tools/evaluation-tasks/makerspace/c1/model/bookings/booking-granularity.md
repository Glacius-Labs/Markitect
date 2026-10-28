---
kind: std/Rule
internal: true
---
# Booking granularity

**Booking granularity is 30 minutes.** A reservation's start and end must each fall on a minute that is a multiple of 30 within the hour (`:00` or `:30`). A reservation lasts at least one granularity step (30 minutes) and at most 240 minutes.
