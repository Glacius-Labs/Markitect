# Agent Rules projection Module

This module produces one provider instruction file from explicitly supplied Core `Schema` and `Definition` values. `Propose` compiles those values with Core, requires exactly one provider-specific `ProjectionPolicy`, and renders only the policy's exact Definition identities plus its supplied guidance.

Codex and Claude files use independent framing. The canonical Schema, Kind, Property, Definition, source provenance, and resolved reference identities are carried as closed JSON facts. Kind names are never used as semantic dispatch. Missing guidance, invalid Core input, uncovered affected identities, unknown target ownership, or an unsafe target path produce escalation.

`TargetObservation` is external evidence supplied by the caller. `Propose` performs no filesystem or provider I/O. It returns a candidate and its SHA-256 digest, and reports `no-op` only when the observed owned file has the same digest. A read-only target that needs a change escalates.

This module proves deterministic representation behavior and its input boundary. It does not establish that an AI agent follows the file or that a provider adopted its instructions.
