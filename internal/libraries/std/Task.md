---
kind: Kind
fields:
  Actor: {required: true, links: std/Stakeholder}
  Steps: {required: true}
  Satisfies: {required: true, links: [std/Goal, std/Constraint, std/Rule, std/Quality, std/Wish]}
parts:
  outcomes: std/Outcome
---
# Task

A Task Definition states one thing the person who runs the system does to it, under `administration/tasks/`. **Actor:** the [[std/Stakeholder]] who does it. **Steps:** numbered, each one act on the machine, with exact names in backticks. **Satisfies:** the [[std/Wish]], [[std/Quality]], [[std/Goal]], [[std/Constraint]] or [[std/Rule]] it exists for.

What holds afterwards are its [[std/Outcome|Outcomes]], under `outcomes/`. What the system does for it by itself is stated by the [[std/Component]] that does it.

Verified when carrying out the steps leaves each Outcome.
