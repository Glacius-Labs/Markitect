---
kind: Kind
---
# Feature

A Feature Definition describes one thing a user or another system can do with the project, as a use case: who does it, what they provide, what happens, and what they get back.

- Start with one sentence on what the Feature achieves.
- Link the [[std/Concept|Concepts]] it reads or changes and the [[std/Rule|Rules]] it must respect.
- Describe the normal outcome and every refusal a caller can observe, with exact codes or messages in backticks.
- When the Feature is reached through an interface, the interface's Definitions, such as a REST Endpoint or a CLI Command, link to the Feature.
