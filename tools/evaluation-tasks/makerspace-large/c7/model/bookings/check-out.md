---
kind: Operation
---
# Check out

Status `completed`; `checkedOutAt` is the request time; `meteredMinutes` is `checkedOutAt − checkedInAt` in minutes, which may be 0 and may extend past the reservation's end or closing time. The following then happen together, in this order:

1. Minutes are drawn from the member's valid packs ([[packs/pack-draw]]). `drawn` is the number drawn, between 0 and `meteredMinutes`, and `billable` is `meteredMinutes − drawn`.
2. A usage [[billing/charge]] is created for the member: `kind` `usage`, `ref` the reservation identifier, `quantity` the billable minutes, `unitPriceCents` the captured rate, `discountPercent` the discount of the member's [[members/tier|tier]] at check-out, `at` the request time. A charge of 0 cents is still created, also when the billable minutes are 0. The reservation's `amountCents` is the charge's amount.
3. The consumables of the machine are drawn from stock ([[consumables/consumption]]). One material charge is created for each consumption row of the machine whose `drawn` is above 0, in ascending `materialId` order, after the usage charge. They use the full metered minutes, whatever the packs drew, and they do not change the reservation's `amountCents`.

Standing is not checked. A lockout of the machine, an unsigned or lapsed waiver and the other rules of check-in do not stop a check-out. The result is the same as without packs and consumables: only the usage charge is returned.

Rules of other modules that apply here: [[billing/charges-and-invoices]], [[packs/pack-draw]] and [[consumables/consumption]].

- **Wire name:** `checkOut`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `reservationId`: identifier, naming an existing [[bookings/reservation]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the status is not `checkedIn`.
- **Result:**
  - `reservation`: a [[bookings/reservation]] record
  - `charge`: a [[billing/charge]] record, the usage charge
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `reservationId`.
