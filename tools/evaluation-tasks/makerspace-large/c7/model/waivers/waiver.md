---
kind: std/Concept
---
# Waiver

The current waiver of one tool family. Publishing a new waiver replaces it, and its version grows by 1.

**Fields**, in the formats of [[values]]:

- `familyId`: identifier, naming an existing [[equipment/family]]
- `version`: count, at least 1 (the first waiver of a family has version 1)
- `text`: text

A record is identified by `familyId`.
