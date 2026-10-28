---
kind: Operation
---
# Revenue report

Summarizes billing for one month:

- `invoicesGenerated`: whether [[billing/invoice|invoices]] for the month have been generated;
- `invoiceCount`: the number of invoices of the month;
- `membershipCents`, `usageCents`, `materialCents`, `courseCents`: the sums of the line amounts of each kind on the invoices of the month;
- `invoicedCents`: the sum of those four;
- `creditAppliedCents`: the sum of `creditAppliedCents` over the invoices of the month;
- `outstandingCents`: the sum of `balanceCents` over the invoices of the month at the request time;
- `paymentsCents`: the sum of `amountCents` of all [[billing/payment|payments]] whose `at` lies in the month, for any invoice, including the parts that became credit;
- `creditsIssuedCents`: the sum of credits issued with `issueCredit` whose request time lies in the month, plus the lockout credits of `lockMachine` requests whose request time lies in the month. Credit from overpayments is not included.

For a month without generated invoices, the invoice figures are 0 and `invoicesGenerated` is `false`; payments and issued credits are still reported.

Rules of other modules that apply here: [[billing/charges-and-invoices]] and [[billing/overdue-invoices]].

- **Wire name:** `revenueReport`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `month`: month
- **Result:**
  - `month`: month
  - `invoicesGenerated`: flag
  - `invoiceCount`: count
  - `membershipCents`: cent amount
  - `usageCents`: cent amount
  - `materialCents`: cent amount
  - `courseCents`: cent amount
  - `invoicedCents`: cent amount
  - `creditAppliedCents`: cent amount
  - `outstandingCents`: cent amount
  - `paymentsCents`: cent amount
  - `creditsIssuedCents`: cent amount
- **Permissions:** staff all, admin all.
