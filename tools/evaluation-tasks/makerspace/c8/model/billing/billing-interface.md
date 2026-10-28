---
kind: Interface
---
# Billing interface

What other modules may use of billing: its operations, the charge, invoice, payment and account records, and the refusal `account_overdue`. Other modules create a charge for a member through billing, and billing computes its amount by [[money]]. Other modules also grant lockout credits through billing and ask whether a member is blocked.

- **Operations:** [[billing/generate-invoices]], [[billing/list-invoices]], [[billing/record-payment]], [[billing/issue-credit]], [[billing/get-account]].
- **Uses:** [[members/members-interface]].
- **Records:** [[billing/charge]], [[billing/invoice]], [[billing/payment]], [[billing/account]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `account_overdue` | The member has an invoice that is more than 30 days overdue. |
| `already_invoiced` | Invoices for the month have already been generated. |
| `invoice_paid` | The invoice is already paid. |
| `month_not_ended` | The month to invoice has not ended at the request time. |
