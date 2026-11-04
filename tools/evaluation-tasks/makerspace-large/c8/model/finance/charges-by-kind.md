---
kind: Operation
---
# Charges by kind

Reports the invoiced charges of a month by kind. The month's invoices are the invoices generated for the month. `rows` has one row for each kind, in the order `membership`, `usage`, `material`, `course`, `deposit`, `pack`, `locker`, `guest`, `lateFee`, `event`, `noShow`, including kinds without lines. For a row:

- `invoicedCents` is the sum of `amountCents` of the lines of that kind on the month's invoices;
- `refundedCents` is the sum of all refunds ([[refunds/refund]]) of those lines, whenever the refunds were issued. A line is identified by its invoice and its position in the invoice's `lines`.

`totalCents` is the sum of `invoicedCents` over the rows, which is the sum of `totalCents` over the month's invoices. For a month without generated invoices every figure is 0.

- **Wire name:** `chargesByKind`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `month`: month
- **Result:**
  - `month`: month
  - `rows`: a list of [[finance/charge-kind-row]] records, one per kind, in the order above
  - `totalCents`: cent amount
- **Permissions:** staff all, admin all.
