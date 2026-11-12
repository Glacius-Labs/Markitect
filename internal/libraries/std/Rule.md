---
kind: Kind
binds: true
fields:
  Applies to:
  Derived from: {links: [std/Goal, std/Constraint, std/Decision, std/Quality, std/Assumption, std/Term, std/ExternalSystem, std/Expectation, std/Rule, std/Concern, std/Wish]}
parts:
  .: std/Rule
---
# Rule

A Rule Definition states something the owner chose that holds across Concepts or Operations, or for the work itself, as a property a reviewer can decide, with exact values in backticks. This includes the definition of a notion and who counts as which role. It lives under `rules/`: of an area or of `domain/` when it is about what the system does, and of `architecture/` or of a Component when it binds how the project is built, tested and run.

**Applies to:** where, when narrower than its place, which is its scope. **Derived from:** what is behind it: a [[std/Goal]], [[std/Constraint]], [[std/Decision]] or [[std/Quality]], a [[std/Concern]] or [[std/Wish]] it answers, an [[std/Assumption]], a [[std/Term]], an [[std/ExternalSystem|External system]] or one of its [[std/Expectation|Expectations]] it rests on, or another Rule. A Rule inside the folder of another Rule is part of it and follows from what that one follows from.

Verified when what it asks for itself is built and nothing that exists contradicts it; where it asks nothing for itself, the places searched for a contradiction are the evidence. What it binds is verified against it item by item.
