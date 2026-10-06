# Agent Rules projection Module

This module produces one provider instruction file from explicitly supplied Core `Schema` and `Definition` values. `Definitions` are eligible for projection; `RelatedDefinitions` are read-only context used to resolve selected references. A provider-specific `ProjectionPolicy` selects exact Definition identities and supplies project-owned guidance.

`Render` returns desired bytes without requiring target observation or writability and performs no I/O. It derives `AGENTS.md` or `CLAUDE.md` under a canonical repository-relative slash `TargetPrefix`; an optional policy target must match that module-owned path. The candidate must fall within an explicit `AllowedRoots` entry.

Codex and Claude files use independent framing. Selected Schema, Kind, and Property purposes, Definition purposes and validated values, source provenance, and resolved reference identities are carried as closed JSON facts. Kind names are never used as semantic dispatch. Missing guidance, invalid Core input, uncovered affected identities, or unsafe paths produce an error from `Render` and escalation from `Propose`.

`TargetObservation` is external evidence supplied by the caller. `Propose` compares the rendered candidate with the exact observed owned path and digest; it reports `no-op` only when those bytes match. A read-only target that needs a change escalates. Neither function reads or writes the target.

This module proves deterministic representation behavior and its input boundary. It does not establish that an AI agent follows the file or that a provider adopted its instructions.
