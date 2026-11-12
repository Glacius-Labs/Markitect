---
kind: Kind
fields:
  Measure:
  Matters to: {links: std/Stakeholder}
parts:
  .: std/Quality
---
# Quality

A Quality Definition states how well the system must do something: what happens, and what the system then does, in one sentence. **Measure:** the value that decides whether it is met, in backticks, with its unit and the load it holds under; or that none is promised.

`quality/` holds one Quality for each characteristic of ISO/IEC 25010, named for it. That one says in one sentence what the characteristic asks of this system, or why it asks nothing, and has no Measure; the Qualities in its folder are its parts. **Matters to:** the [[std/Stakeholder|Stakeholders]] who feel it. `quality/quality` names the characteristics that decide the design, the most important first.

Verified when a test or a review shows that the Measure is met. A Quality that names a characteristic is verified when its parts are, and one that asks nothing when nothing in the project contradicts that.
