---
kind: Kind
fields:
  Reaches: {required: true, links: std/Operation}
parts:
---
# Route

A Route Definition states how one Operation is reached through an [[std/Interface]]: the method and path of an API, or the command and its arguments at a command line, with the exact names in backticks. **Reaches:** the [[std/Operation]]. It lives under the Interface's `routes/` and is named for the Operation.

Verified when the Operation is reachable as stated.
