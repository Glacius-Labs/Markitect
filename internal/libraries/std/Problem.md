---
kind: Kind
realized: false
links: [std/Term, std/Constraint, std/Assumption, std/ExternalSystem, std/Expectation, std/Stakeholder, std/Problem]
fields:
  Affects: {required: true, links: std/Stakeholder}
  Cost:
parts:
---
# Problem

A Problem Definition states one thing that goes wrong or costs today, and for whom. It lives under `purpose/problems/` and links only what lies under `context/`, Stakeholders and other Problems.

**Affects:** the [[std/Stakeholder|Stakeholders]] it goes wrong for. **Cost:** how often it happens or how much it costs, where that is known.

A [[std/Goal]] that removes it names it in its **Removes** field; the Problem does not link the Goal.

Verified when nothing in the project makes it worse.
