---
kind: Operation
---
# Record payment

`appliedCents` is the smaller of `amountCents` and the invoice's `balanceCents`; `creditedCents` is `amountCents − appliedCents`. The invoice's `paidCents` increases and its `balanceCents` decreases by `appliedCents`; when `balanceCents` reaches 0 its status becomes `paid`. The member's credit balance increases by `creditedCents`. A payment record is created with the next payment identifier, `memberId` the invoice's member (whoever acts) and `at` the request time. Paying after the due date is allowed. The `invoice` result is the invoice after the change; `creditBalanceCents` is the member's credit balance after the change.

- **Wire name:** `recordPayment`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `invoiceId`: identifier, naming an existing [[billing/invoice]]
  - `amountCents`: integer from `1` to `100000000` cents (an amount paid or credited in one request)
- **Refusals** at check 7, in this order:
  1. `invoice_paid` when the invoice's status is `paid`.
- **Result:**
  - `payment`: a [[billing/payment]] record
  - `invoice`: a [[billing/invoice]] record
  - `creditBalanceCents`: cent amount
- **Permissions:** member own, staff all, admin all.
- **Audit target:** the new `paymentId`.
