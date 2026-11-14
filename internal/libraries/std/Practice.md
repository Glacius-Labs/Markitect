---
kind: Kind
realized: false
links: [std/Term, std/Constraint, std/Assumption, std/ExternalSystem, std/Expectation, std/Stakeholder]
fields:
  Steps: {required: true}
  Problems: {links: std/Problem}
parts:
---
# Practice

A Practice Definition states how one [[std/Stakeholder]] does one thing today, without the system. It is a part of the Stakeholder, under `today/` in the folder of its name, and links only what lies under `context/` and Stakeholders.

**Steps:** numbered, each one act, as it is done today. **Problems:** the [[std/Problem|Problems]] that arise in it.

It describes the world as it is and does not bind.

Verified by nothing; it is read, not checked.
