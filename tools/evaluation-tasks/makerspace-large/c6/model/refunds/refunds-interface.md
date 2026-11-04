---
kind: Interface
---
# Refunds interface

What other modules may use of refunds: its operations, the refund record and the refusals below. Finance also asks how much has been refunded on a line of an invoice.

- **Operations:** [[refunds/issue-refund]], [[refunds/list-refunds]].
- **Uses:** [[members/members-interface]], [[billing/billing-interface]], [[ledger/ledger-interface]].
- **Records:** [[refunds/refund]].
- **Capabilities** that are not operations:
  - **Refunded on a line.** The caller names an invoice and a line number. The answer is the sum of `amountCents` of all refunds of that invoice and line, and 0 when there is none.

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `invalid_line` | The invoice has no line with the given number. |
| `invoice_not_paid` | The invoice's status is not `paid`. |
| `refund_too_large` | The amount is more than what is left of the line after its earlier refunds. |
