---
paths:
  - realization/billing/**
---
# Registry: billing

| File | Realizes | Why |
|---|---|---|
| `realization/billing/billing.go` | [[billing/billing-interface]] [[billing/charge]] [[billing/invoice]] [[billing/payment]] [[billing/charges]] [[billing/invoices]] | Charges, invoices and payments, what other modules may ask of billing and the credit it grants |
| `realization/billing/blocks.go` | [[billing/due-date-and-block]] | The block of a member with an invoice more than 30 days overdue |
| `realization/billing/invoices.go` | [[billing/generate-invoices]] [[billing/issuing-invoices]] [[billing/charges]] | Invoice generation for a month |
| `realization/billing/operations.go` | [[billing/list-invoices]] [[billing/record-payment]] [[billing/issue-credit]] [[billing/get-account]] [[billing/account]] [[billing/paying-invoices]] [[billing/credit]] | Billing operations: invoices, payments, credit and accounts |
| `realization/billing/views.go` | [[billing/charge]] [[billing/invoice]] [[billing/payment]] | JSON views of the charge, invoice and payment records |
