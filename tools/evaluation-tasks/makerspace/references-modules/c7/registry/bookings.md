---
paths:
  - realization/bookings/**
---
# Registry: bookings

| File | Realizes | Why |
|---|---|---|
| `realization/bookings/bookings.go` | [[bookings/bookings-interface]] [[bookings/reservation]] [[bookings/reservations]] [[bookings/reservation-rules]] [[bookings/metering]] [[bookings/lockout]] [[bookings/lockouts]] | Reservations and lockouts, and what other modules may ask of bookings |
| `realization/bookings/hours.go` | [[bookings/opening-hours]] [[bookings/opening-hours-of-day]] [[bookings/booking-granularity]] | Opening hours of a day and the booking granularity and duration constants |
| `realization/bookings/operations.go` | [[bookings/create-reservation]] [[bookings/cancel-reservation]] [[bookings/check-in]] [[bookings/check-out]] [[bookings/list-member-reservations]] [[bookings/get-machine-schedule]] [[bookings/lock-machine]] [[bookings/reservations]] [[bookings/reservation-rules]] [[bookings/metering]] [[bookings/booking-granularity]] [[bookings/lockouts]] | Booking operations: reservations, check-in, check-out, member reservations, machine schedules and lockouts |
| `realization/bookings/views.go` | [[bookings/reservation]] [[bookings/lockout]] | JSON views of the reservation and lockout records |
