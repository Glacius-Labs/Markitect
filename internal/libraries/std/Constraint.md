---
kind: Kind
binds: true
realized: false
links: [std/Term, std/Constraint, std/Assumption, std/ExternalSystem, std/Expectation]
fields:
  Source:
  Applies to:
parts:
  .: std/Constraint
---
# Constraint

A Constraint Definition states something imposed that restricts what the system may do, so that a reviewer can decide whether the project respects it: by law, by the business, by another system, by nature or by a technical given. It lives under `context/constraints/` and links only what lies under `context/`; what the owner chooses is a [[std/Rule]].

**Source:** who or what imposes it. **Applies to:** where, in words, when narrower than its place. A Constraint inside the folder of another has that one's source.

Verified when what it asks for itself is built and nothing that exists contradicts it; where it asks nothing for itself, the places searched for a contradiction are the evidence. What it binds is verified against it item by item.
