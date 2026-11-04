---
kind: std/Concept
---
# Order status

The step an [[purchasing/order]] has reached.

| Value | Meaning |
| --- | --- |
| `draft` | The order is being written; lines can be added. |
| `submitted` | The order waits for approval. |
| `approved` | The order is approved; stock can be received against its lines. |
| `received` | Every line is fully received. |
