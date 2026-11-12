---
kind: Kind
fields:
  When: {required: true}
  Satisfies: {required: true, links: [std/Goal, std/Constraint, std/Rule, std/Quality, std/Wish]}
parts:
  outcomes: std/Outcome
---
# Reaction

A Reaction Definition states something the system does by itself, with no caller, in the Package of its area, under `reactions/`. **When:** what has happened or which time has come, in terms of [[std/Concept|Concepts]] and their states. **Satisfies:** the [[std/Goal]], [[std/Constraint]], [[std/Rule]], [[std/Quality]] or [[std/Wish]] it exists for.

What then holds, and no later than when, are its [[std/Outcome|Outcomes]] under `outcomes/`; where it does nothing in a case a reader could expect, an Outcome says so. It is never stated as steps.

Verified when each Outcome is shown to hold within its limit for each case of **When**.
