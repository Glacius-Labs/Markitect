---
kind: Kind
last: true
fields:
  Actor: {required: true, links: std/Stakeholder}
  Steps: {required: true}
parts:
  outcomes: std/Outcome
---
# Scenario

A Scenario Definition describes one path a [[std/Stakeholder]] takes through the system, under `scenarios/`. It is a check and specifies nothing: where it and an [[std/Operation]] differ, the Operation holds. **Actor:** the Stakeholder. **Steps:** numbered, each naming the Operation performed; each step is allowed by what the steps before it leave behind.

What holds at the end are its [[std/Outcome|Outcomes]], under `outcomes/`.

Verified when it has been run: a test runs what the project delivers, the built program where it delivers one, started the way the spec says it is started, and takes each step as the Actor would, through the interface the Actor uses, and each Outcome holds at the end. A test that wires parts of the system together inside itself verifies no Scenario, and neither does reading the code.
