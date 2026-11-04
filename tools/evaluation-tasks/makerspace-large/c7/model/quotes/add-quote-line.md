---
kind: Operation
---
# Add quote line

Adds a [[quotes/quote-line]] to the quote, after its other lines, with `unitPriceCents` `null`. Nothing else of the quote changes. A line is never changed or removed.

- **Wire name:** `addQuoteLine`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `quoteId`: identifier, naming an existing [[quotes/quote]]
  - `lineKind`: `machine` or `material`
  - `ref`: identifier; it is judged by the second refusal below, not at check 5
  - `quantity`: integer from `1` to `100000` (minutes of machine time for a `machine` line, units for a `material` line)
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the quote's status is not `draft`.
  2. `unknown_line_ref` when `lineKind` is `machine` and no machine has the identifier `ref`, or `lineKind` is `material` and no material has it.
- **Result:**
  - `quote`: a [[quotes/quote]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `quoteId`.
