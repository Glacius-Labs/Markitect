---
kind: Operation
---
# Get account

Returns the member's account: `creditBalanceCents` is the member's credit balance; `openBalanceCents` is the sum of `balanceCents` over the member's invoices with status `open`; `overdueBalanceCents` is the same sum over the member's overdue invoices; `unbilledCents` is the sum of the amounts of the member's unbilled charges; `blocked` states whether the member is blocked at the request time.

- **Wire name:** `getAccount`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `memberId`: identifier, naming an existing [[members/member]]
- **Result:**
  - `account`: a [[billing/account]] record
- **Permissions:** member own, staff all, admin all.
