---
kind: Interface
---
# Billing interface

What other modules may use of billing: its operations and the charge, invoice, payment and account records. Other modules create a charge for a member through billing; billing computes its amount by [[money]]. Other modules also use the capabilities listed below.

- **Operations:** [[billing/generate-invoices]], [[billing/list-invoices]], [[billing/record-payment]], [[billing/issue-credit]], [[billing/get-account]].
- **Uses:** [[members/members-interface]], [[ledger/ledger-interface]], [[households/households-interface]].
- **Records:** [[billing/charge]], [[billing/invoice]], [[billing/payment]], [[billing/account]].
- **Capabilities** that are not operations:
  - **Create a charge.** The caller names the member and gives the `kind` (any kind of [[billing/charge]]), `ref`, `quantity`, `unitPriceCents`, `discountPercent` and `at`. Billing computes `amountCents` by [[money]], fixes it, and keeps the charge unbilled until an invoice takes it ([[billing/charges-and-invoices]]). It returns the charge. A charge of 0 cents is created like any other. Billing refuses nothing; the caller has checked what its own operation requires.
  - **Add credit.** The caller names the member, an amount in cents, the debit account and the memo. The member's credit balance increases by the amount, and the entry that debits the named account and credits `creditLiability` by the amount, with that memo and `at` the request time, is posted ([[ledger/postings]]). An amount of 0 changes nothing and posts nothing. The credit does not count as a credit issued with [[billing/issue-credit]].
  - **Open invoices.** All invoices whose status is `open` at a request time, in ascending invoice number.
  - **Whether a charge is on a paid invoice.** The caller names a charge by its `kind` and `ref`. The answer is yes when a line with that `kind` and `ref` is on an invoice whose status is `paid`, and no when no invoice has such a line or the invoice is `open`.
  - **An invoice.** An invoice by its identifier, with its lines in the order of the invoice. The invoice a member has for a month, if any: a member has at most one invoice for a month. The invoices of a month, in ascending invoice number, which is creation order.
  - **A member's account.** The figures of [[billing/account]]: credit balance, open balance, overdue balance and unbilled amount.
  - **Overdue invoices.** All invoices that are overdue at a request time ([[billing/overdue-invoices]]), in ascending invoice number. The invoice number is the number `N` of `inv-N`, compared as a number.
  - **Whether a month is invoiced.** Whether [[billing/generate-invoices]] has been accepted for the month.
  - **Payments and credits of a month.** The sum of `amountCents` of all [[billing/payment|payments]] whose `at` lies in the month, and the sum of the credits issued with [[billing/issue-credit]] whose request time lies in the month.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_invoiced` | Invoices for the month have already been generated. |
| `invoice_paid` | The invoice is already paid. |
| `month_not_ended` | The month has not ended at the request time. |
