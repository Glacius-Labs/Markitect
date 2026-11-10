---
kind: Kind
---
# Feature

A Feature Definition describes one thing a user or another system can do with the project, as a use case: who does it, what they provide, what happens, and what they get back.

- Start with one sentence on what the Feature achieves.
- Link what it relies on: the [[std/Concept|Concepts]] it reads or changes, and a [[std/Rule]] whose statement its text uses. A Rule of its Package or of a Package above binds it by its place and needs no link.
- Describe the normal outcome and every refusal a caller can observe, each with its exact code in backticks. State the wording of a message only when a caller relies on it.
- When the Feature is reached through an interface, the interface's Definitions, such as a REST Endpoint or a CLI Command, link to the Feature.
