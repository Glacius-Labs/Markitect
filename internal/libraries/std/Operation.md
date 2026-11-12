---
kind: Kind
fields:
  Who: {required: true}
  Given:
  Checked in this order: {links: std/Condition}
  Returns:
  Satisfies: {required: true, links: [std/Goal, std/Constraint, std/Rule, std/Quality, std/Wish]}
parts:
  conditions: std/Condition
  outcomes: std/Outcome
  results: std/Result
---
# Operation

An Operation Definition states one thing a caller can do to a [[std/Concept]] or ask of it, in the Package of that Concept, under `operations/`. It says what it achieves, then **Who** may (roles in backticks), what is **Given**, saying which of it may be left out, what it **Returns** and what it **Satisfies**: a [[std/Goal]], [[std/Constraint]], [[std/Rule]], [[std/Quality]] or a [[std/Wish]].

**Checked in this order:** its [[std/Condition|Conditions]], numbered by place, each with the refusal code that answers it; the first that fails is reported, and Conditions at one place are named together. Its own Conditions, its [[std/Outcome|Outcomes]] and its [[std/Result|Results]] lie under `conditions/`, `outcomes/` and `results/`. Without Conditions every caller that **Who** names is allowed; without Outcomes it changes nothing.

Verified when only the callers that **Who** names are allowed, each Condition, Outcome and Result is shown, and what it returns is what **Returns** says.
