---
kind: Kind
realized: false
links: [std/Term, std/Constraint, std/Assumption, std/ExternalSystem, std/Expectation, std/Goal, std/Stakeholder]
fields:
  Result: {required: true}
  Changes: {required: true}
  Obstacle: {required: true}
parts:
---
# Goal

A Goal Definition states a result the owner wants from the system. It lives under `purpose/goals/` and is stated without saying how the system achieves it.

Its fields, in this order:

- **Result:** what is to be achieved, measurable where it can be.
- **Changes:** what is different in the world once the system is in use.
- **Obstacle:** what stands in the way today.

A Goal links only what lies under `context/`, other Goals and the [[std/Stakeholder|Stakeholders]] it is for. What satisfies a Goal links it; the Goal does not link what satisfies it.

Verified when what names it in a **Satisfies** field exists and is verified, and together that can bring about the result.
