---
kind: Interface
---
# Billing interface

What other modules may use of billing: its operations and the charge, invoice, payment and account records. Other modules create a charge for a member through billing; billing computes its amount by [[money]].

- **Operations:** [[billing/generate-invoices]], [[billing/list-invoices]], [[billing/record-payment]], [[billing/issue-credit]], [[billing/get-account]].
- **Records:** [[billing/charge]], [[billing/invoice]], [[billing/payment]], [[billing/account]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `already_invoiced` | Invoices for the month have already been generated. |
| `invoice_paid` | The invoice is already paid. |
| `month_not_ended` | The month to invoice has not ended at the request time. |
