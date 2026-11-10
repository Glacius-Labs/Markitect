---
kind: Kind
---
# Concept

A Concept Definition explains one thing of the project's domain: what it is, what it consists of, how it relates to other Concepts, and which states it can be in.

- Name the attributes a caller relies on and their meaning; exact values go in backticks.
- Link another Concept where this one relies on what it states, from the specific to the general: a loan links the member it relies on, or only the member's identifier when that is all it uses, and the member does not link its loans.
- State the invariants that always hold for it. Rules that hold across several Concepts are [[std/Rule|Rules]] of their own.
- Leave data structures, storage and algorithms to the realization; describe what good documentation of the domain would describe.
