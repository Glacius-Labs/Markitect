---
kind: Operation
---
# Finance summary

Summarizes the money of one month, calculated at the request time:

- `invoicedCents`: the sum of `amountCents` of all lines of all kinds on the invoices of the month;
- `refundedCents`: the sum of all refunds ([[refunds/refund]]) of lines of those invoices, whenever the refunds were issued;
- `paymentsCents`: the sum of `amountCents` of all [[billing/payment|payments]] whose `at` lies in the month, as in `revenueReport`;
- `creditsIssuedCents`: the sum of credits issued with [[billing/issue-credit]] whose request time lies in the month, as in `revenueReport`;
- `journalEntries`: the number of [[ledger/journal-entry|journal entries]] whose `at` lies in the month;
- `trialBalanced`: true when the total of all debits of the whole journal equals the total of all credits, which [[ledger/postings|the postings of the ledger]] make true at all times.

For a month without generated invoices, `invoicedCents` and `refundedCents` are 0.

- **Wire name:** `financeSummary`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `month`: month
- **Result:**
  - `month`: month
  - `invoicedCents`: cent amount
  - `refundedCents`: cent amount
  - `paymentsCents`: cent amount
  - `creditsIssuedCents`: cent amount
  - `journalEntries`: count
  - `trialBalanced`: flag
- **Permissions:** admin all.
