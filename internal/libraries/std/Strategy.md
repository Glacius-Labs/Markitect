---
kind: Kind
realized: false
fields:
  Satisfies: {links: [std/Quality, std/Goal]}
  Carried out by: {links: [std/Decision, std/Rule]}
parts:
  .: std/Strategy
---
# Strategy

A Strategy Definition states in one sentence one fundamental way in which the design reaches what is asked of it, under `architecture/strategy/`. **Satisfies:** the [[std/Quality|Qualities]] or [[std/Goal|Goals]] it reaches. **Carried out by:** the [[std/Decision|Decisions]] and [[std/Rule|Rules]] that carry it out.

`architecture/strategy/strategy` states the design as a whole in one sentence; the Strategies in its folder are its parts.

Verified when what **Carried out by** names exists and nothing in the project works against it.
