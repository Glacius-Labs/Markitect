---
kind: Kind
fields:
parts:
  attributes: std/Attribute
  states: std/State
  invariants: std/Invariant
  conditions: std/Condition
---
# Concept

A Concept Definition says in one sentence what one thing the system manages is. It lives in the Package of its area, under `concepts/`, and links the Concepts it relies on, in its sentence or through its parts.

Its [[std/Attribute|Attributes]], [[std/State|States]] and [[std/Invariant|Invariants]] lie under `attributes/`, `states/` and `invariants/`; [[std/Condition|Conditions]] that several Operations on it check lie under `conditions/`. A Concept does not list its Operations.

Verified when its parts are and what its sentence states holds.
