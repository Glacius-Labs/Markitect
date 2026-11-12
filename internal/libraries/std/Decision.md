---
kind: Kind
binds: true
fields:
  Reason:
parts:
  consequences: std/Consequence
  alternatives: std/Alternative
---
# Decision

A Decision Definition records a choice the project made and must keep, in one sentence. It lives under `decisions/`, of `architecture/` or of the Package it binds. **Reason:** why; this field does not bind.

Its [[std/Consequence|Consequences]] and rejected [[std/Alternative|Alternatives]] live in the folder of its name, under `consequences/` and `alternatives/`. Its place is its scope; a Definition whose text relies on it links it.

Verified when its Consequences are followed and nothing contradicts it.
