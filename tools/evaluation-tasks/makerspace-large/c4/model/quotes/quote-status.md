---
kind: std/Concept
---
# Quote status

The step a [[quotes/quote]] has reached.

| Value | Meaning |
| --- | --- |
| `draft` | The quote is being written; lines can be added. |
| `sent` | The quote is priced and sent to the customer; it waits for a decision. |
| `accepted` | The customer accepted the quote; a job can start from it. |
| `rejected` | The customer rejected the quote. |
