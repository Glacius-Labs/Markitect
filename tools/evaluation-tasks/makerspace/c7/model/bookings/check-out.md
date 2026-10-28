---
kind: Operation
---
# Check out

Status `completed`; `checkedOutAt` is the request time; `meteredMinutes` is `checkedOutAt − checkedInAt` in minutes, which may be 0 and may extend past the reservation's end or closing time. A usage [[billing/charge]] is created for the member: `kind` `usage`, `ref` the reservation identifier, `quantity` the metered minutes, `unitPriceCents` the captured rate, `discountPercent` the discount of the member's [[members/tier|tier]] at check-out, `at` the request time. A charge of 0 cents is still created. The reservation's `amountCents` is the charge's amount. Standing is not checked.

Rules of other modules that apply here: [[billing/charges]].

- **Wire name:** `checkOut`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `reservationId`: identifier, naming an existing [[bookings/reservation]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the status is not `checkedIn`.
- **Result:**
  - `reservation`: a [[bookings/reservation]] record
  - `charge`: a [[billing/charge]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `reservationId`.
