# Projection binding notes

This is a structural/binding slice, not a complete controller-start fixture: its Markdown Projection does not select per-Kind Markdown policies and correctly escalates when planning that representation. The [complete Classic Commerce example](../classic-commerce/README.md) supplies those policies and its own public build/behavior check and native CLI walkthrough. No internal test preparation is needed for that example.

## Finite first-slice scope

A Projection declares the exact ordered set of canonical Definition identities to include, the desired representation (`dotnet` or `markdown`), the repository and target path, and zero or more typed ProjectionPolicy references. Runtime project configuration supplies one exact `ProjectionBinding` from this Projection identity to an installed Projection Module name. The Host rejects a missing or duplicate binding, a Schema Module, an ambiguous Module entrypoint, or a target that does not match the canonical representation. Switching between compatible Modules changes the runtime request and operational pin while leaving canonical intent and its Model digest unchanged. The Host resolves every listed identity against the compiled model, rejects missing or duplicate identities, and includes a reference edge only when both endpoints are in the declared set. An edge leaving that set remains an external dependency or gap; it does not silently enlarge scope.

Scope identity objects are ordinary closed objects rather than Core reference values. A Core reference Property requires one exact target Kind, which would couple Foundation to each adopting ontology. Core validates each identity object's shape; Host binding resolves it against the compiled model. The runtime `ProjectionBinding` identifies a Module by its exact name; catalog activation pins its exact version and digest. The selected Projection Module must have exactly one registered execution entrypoint; ambiguous registrations fail instead of selecting the first. Any internal entrypoint id, version, and package pin are operational tool provenance, not canonical Kinds or Projection nouns.

ProjectionPolicy.sourceKind uses kindReference to identify a Kind from any explicitly supplied Schema without selecting all its Definitions. The .NET fixture supplies a separate policy for each selected Kind. A policy does not inherit to other Kinds.

## Strict Module types and one execution loop

The Foundation and commerce packages are Schema Modules: they register Schemas and no projection capability. The .NET and Markdown packages are Projection Modules: they register target execution capabilities and no Schemas. Projection Modules do not add Kinds, and Schema Modules do not depend on target Modules.

Each Projection follows one Executor-to-candidate-to-independent-Verifier loop. The .NET target has per-Kind project guidance. The Markdown Module describes a faithful source-view contract; a deterministic Markdown renderer may be an internal Executor tool, not a parallel projection architecture. Its output remains a representation and never becomes semantic authority for Definitions or another Projection.

## Custom Kind pressure test

The commerce Schema defines a project-specific EffectAxis Kind that is not listed as a supported Kind in the .NET Module. Its selected Definition has a typed field and explicit ProjectionPolicy guidance. The static fixture and descriptor shape can show that the Projection Module does not enumerate a fixed ontology matrix and the Core can structurally compile this input. They do not show that an AI Executor interprets the Kind safely, produces a semantically adequate candidate, or improves real work. Those claims require an executed, independently verified candidate.

The .NET target is the current repository (.) and src/; the documentation target is docs/represented/. No general graph closure language is established here. ProjectionRequest, ProjectionResult, and immutable ProjectionRecord remain operational Host contracts, separate from canonical Projection intent.
