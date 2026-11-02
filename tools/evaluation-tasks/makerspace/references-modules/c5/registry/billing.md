---
paths:
  - realization/billing/**
---
# Registry: billing

| File | Realizes | Why |
|---|---|---|
| `realization/billing/billing.go` | [[billing/billing-interface]] [[billing/charge]] [[billing/invoice]] [[billing/payment]] [[billing/charges-and-invoices]] [[billing/overdue-invoices]] | Charges, invoices and payments, what other modules may ask of billing and the credit it grants |
| `realization/billing/blocks.go` | [[billing/account-blocks]] | The block of a member with an invoice more than 30 days overdue |
| `realization/billing/invoices.go` | [[billing/generate-invoices]] [[billing/charges-and-invoices]] | Invoice generation for a month |
| `realization/billing/operations.go` | [[billing/list-invoices]] [[billing/record-payment]] [[billing/issue-credit]] [[billing/get-account]] [[billing/account]] [[billing/payments-and-credit]] | Billing operations: invoices, payments, credit and accounts |
| `realization/billing/views.go` | [[billing/charge]] [[billing/invoice]] [[billing/payment]] [[billing/overdue-invoices]] | JSON views of the charge, invoice and payment records |
