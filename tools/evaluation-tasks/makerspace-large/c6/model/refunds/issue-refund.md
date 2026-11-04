---
kind: Operation
---
# Issue refund

Refunds `amountCents` of one line of the paid invoice. The line is the one at position `lineNumber` in the invoice's `lines`; position 1 is the membership line. A refund is created with the next refund identifier and `at` the request time. The credit balance of the invoice's member increases by `amountCents`, and the entry is posted as [[ledger/postings|the postings of the ledger]] state. `creditBalanceCents` is the member's credit balance after the change. The invoice, its lines and its status do not change. The member's standing is not checked.

The **left** of a line is the line's `amountCents` less the sum of `amountCents` of the earlier refunds of the same invoice and line. A line can be refunded in several parts until nothing is left; a line of 0 cents has nothing left.

Rules of other modules that apply here: [[ledger/postings]].

- **Wire name:** `issueRefund`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `invoiceId`: identifier, naming an existing [[billing/invoice]]
  - `lineNumber`: integer from `1` to `1000` (the position of a line in an invoice; 1 is the first line)
  - `amountCents`: integer from `1` to `100000000` cents (a refund amount)
  - `reason`: text
- **Refusals** at check 7, in this order:
  1. `invoice_not_paid` when the invoice's status is not `paid`.
  2. `invalid_line` when `lineNumber` is greater than the number of lines of the invoice.
  3. `refund_too_large` when `amountCents` is greater than what is left of the line. An amount equal to what is left is allowed.
- **Result:**
  - `refund`: a [[refunds/refund]] record
  - `creditBalanceCents`: cent amount
- **Permissions:** admin all.
- **Audit target:** the new `refundId`.
