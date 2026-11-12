---
kind: Kind
realized: false
links: [std/Term, std/Constraint, std/Assumption, std/ExternalSystem, std/Expectation]
fields:
  Ensured by: {required: true}
  Therefore not built: {required: true}
parts:
---
# Assumption

An Assumption Definition states something the system takes for granted so that it can be simpler. It lives under `context/assumptions/`. Nobody builds it; somebody else has to make it true.

It has, in this order:

- The statement.
- **Ensured by:** who has to make it true.
- **Therefore not built:** what the system does not do because of it.

It links only what lies under `context/`.

Verified when nothing in the project contradicts it and the project builds none of what **Therefore not built** names.
