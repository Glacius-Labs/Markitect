---
kind: Kind
realized: false
links: [std/Term, std/Constraint, std/Assumption, std/ExternalSystem, std/Expectation, std/Stakeholder, std/Concern]
fields:
  Between: {required: true, links: std/Concern}
  Settled by: {required: true, links: [std/Goal, std/Constraint, std/Quality, std/Decision]}
parts:
---
# Conflict

A Conflict Definition states two or more [[std/Concern|Concerns]] of different [[std/Stakeholder|Stakeholders]] that the system cannot all serve fully, and who gives way, in one sentence. It lives under `purpose/conflicts/`.

**Between:** the Concerns, two or more. **Settled by:** the [[std/Goal]], [[std/Constraint]], [[std/Quality]] or [[std/Decision]] that decides it; it names only what lies at Root, since Root may not link down into a child Package.

Verified when what **Settled by** names exists and the project decides the case as the Conflict states.
