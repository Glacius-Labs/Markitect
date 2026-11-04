---
kind: Operation
---
# Return tool

The loan gets status `returned` and `returnedAt` the request time. The tool's `onLoan` becomes `false`. When the loan is overdue on the date of the request time and its days overdue are more than its `chargedDays`, a late-fee charge is created ([[loans/late-fee-charge]]). `charge` is that charge, otherwise `null`. Standing is not checked, no shift is needed and no notice is posted.

Rules of other modules that apply here: [[billing/charges-and-invoices]].

- **Wire name:** `returnTool`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `loanId`: identifier, naming an existing [[loans/loan]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the loan's status is `returned`.
- **Result:**
  - `loan`: a [[loans/loan]] record, after the change
  - `tool`: a [[loans/tool]] record, after the change
  - `charge`: a [[billing/charge]] record, or `null`
- **Permissions:** staff all, admin all.
- **Audit target:** `loanId`.
