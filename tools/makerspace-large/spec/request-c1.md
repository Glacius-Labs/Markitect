# Change request: Bookings in 15-minute steps

Most jobs on the small laser take a quarter of an hour, yet members have to block half an hour and the machine stands idle for the rest. Please make the booking granularity 15 minutes.

- A reservation's `start` and `end` must each fall on a minute that is a multiple of 15 within the hour: `:00`, `:15`, `:30` or `:45`.
- A reservation lasts at least 15 minutes and, as today, at most 240 minutes.
- `createReservation` keeps its refusal codes and their order. `invalid_slot` is now answered when `start` or `end` is not on a 15-minute boundary or the length is outside 15 to 240 minutes.

Everything else about reservations stays as it is.
