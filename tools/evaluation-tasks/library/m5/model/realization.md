---
kind: std/Rule
---
# Realization

Across the [[project|library]], deliver a callable local implementation of the operations concerning [[members]], [[catalog|copies]], [[lending|loans]] and [[lending/reminders|overdue assessment]], automated tests and concise usage documentation. These delivery obligations apply to every listed area.

- Tests cover successful operations, expected refusals, borrowing-limit and due-date boundaries for both loan classes, renewals and repeated returns.
- Documentation explains how to run and test the implementation and its scope.
- Code, tests and documentation must agree with the model.

No user interface, deployment or production database is required. The [[callable-interface]] fixes the program boundary, language, build and test commands and wire format. Frameworks, internal source layout and domain design remain realization choices; no architecture is inferred from the historical library sketch.
