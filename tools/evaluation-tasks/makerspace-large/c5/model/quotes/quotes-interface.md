---
kind: Interface
---
# Quotes interface

What other modules may use of quotes: its operations, the customer, quote and quote line records and the refusal codes below. Other modules also ask quotes, for a quote identifier, whether the quote exists and for its customer, its status, its `totalCents` and its lines.

- **Operations:** [[quotes/add-customer]], [[quotes/create-quote]], [[quotes/add-quote-line]], [[quotes/send-quote]], [[quotes/decide-quote]], [[quotes/get-quote]].
- **Uses:** [[equipment/equipment-interface]], [[materials/materials-interface]].
- **Records:** [[quotes/customer]], [[quotes/quote]], [[quotes/quote-line]].

The refusal codes of this module:

| Code | Meaning |
| --- | --- |
| `empty_quote` | The quote has no line. |
| `quote_expired` | The quote is accepted after the date it is valid until. |
| `unknown_line_ref` | The line names no machine or no material. |
