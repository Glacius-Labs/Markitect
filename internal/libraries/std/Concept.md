---
kind: Kind
---
# Concept

A Concept Definition explains one thing of the project's domain: what it is, what it consists of, how it relates to other Concepts, and which states it can be in.

- Name the attributes a caller relies on and their meaning; exact values go in backticks.
- Link the Concepts it relates to, by path.
- State the invariants that always hold for it. Rules that hold across several Concepts are [[std/Rule|Rules]] of their own.
- Leave data structures, storage and algorithms to the realization; describe what good documentation of the domain would describe.
