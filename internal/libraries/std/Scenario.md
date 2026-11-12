---
kind: Kind
fields:
  Actor: {required: true, links: std/Stakeholder}
  Steps: {required: true}
parts:
  outcomes: std/Outcome
---
# Scenario

A Scenario Definition describes one path a [[std/Stakeholder]] takes through the system, under `scenarios/`. It is a check and specifies nothing: where it and an [[std/Operation]] differ, the Operation holds. **Actor:** the Stakeholder. **Steps:** numbered, each naming the Operation performed; each step is allowed by what the steps before it leave behind.

What holds at the end are its [[std/Outcome|Outcomes]], under `outcomes/`.

Verified when it runs through as stated.
