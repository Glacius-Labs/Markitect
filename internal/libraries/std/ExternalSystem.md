---
kind: Kind
realized: false
links: [std/Term, std/Constraint, std/Assumption, std/ExternalSystem, std/Expectation]
fields:
  Uses: {required: true}
  When it fails: {required: true}
parts:
  expectations: std/Expectation
---
# External system

An External system Definition says in one sentence what another system that ours talks to and does not build is, and who runs it. It lives under `context/external-systems/` and links only what lies under `context/`.

**Uses:** what our system uses of it, with exact names where a caller must match them. **When it fails:** what holds on our side when it refuses, is slow or is away. Its [[std/Expectation|Expectations]] live in the folder of its name, under `expectations/`.

Verified when the project uses no more of it than stated and behaves as stated when it fails.
