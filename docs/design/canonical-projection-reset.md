# Canonical projection reset

Status: accepted owner-directed migration, 2026-10-05. Starting source main: `b03369342065d41b04d4b0feed4a126caf953b40`. Published v0.13.0 remains immutable. The owner invoked the reset and operating model, then explicitly required reuse/generalization of the validated projection-first implementation. Original inputs are preserved under `docs/strategy/sources/canonical-reset/`; this decision owns implementation choices rather than making all illustrative syntax mandatory.

## Architecture delta

| Existing behavior | Decision |
|---|---|
| Fixed source acquisition, opaque snapshots, digests and materialization | Keep. Source identity is supplied to compilation; Core does not call Git. |
| Typed vocabulary, normalized IR and explicit reference graph | Generalize into Schema, Kind, Property and Definition. Full API version participates in identity. |
| Domain relations, finite constraints, PolicyResults and exceptions | Isolate the exact historical contract outside the new structural Core. Preserve its deterministic behavior, digests, staleness, diagnostics and tests. It is not obsolete. |
| Projection contracts, Plan/Apply/Verify, immutable checks, provenance and drift/impact | Reuse and migrate through explicit adapters/mechanics. Do not replace with a parallel competing engine or weaken trust boundaries. |
| Compiled Go capability packages | Distinguish implementation packages from user-installable Module packages. Their existing behavior remains evidence; current package locations are not a semantic promise. |
| Foundation and architecture vocabulary | Versioned schema-only Module packages; no compiler built-ins. |
| Markdown and code representation | Independently registered Projectors consume canonical data and their own supplied target state. No projector consumes another projector's representation as semantic authority. |
| Projection/run evidence | Separate immutable operational records and verification results; neither is canonical desired intent. |
| Agent/provider output | Thin entrypoints to canonical scope and operation contracts. No inferred adoption or reviewer authentication. |

## Target ownership

Core (`internal/core`) owns structural compilation, qualified definition/kind identity, closed typed values, property multiplicity, resolved reference/kind-reference facts, normalized model, source provenance and diagnostics. Generic snapshot values remain reusable. Core owns no policy assertions, engineering vocabulary, Project configuration, output paths, providers, execution or persistence.

Host owns source codecs, Module manifest/catalog activation, explicit source selection, compatibility lowering, context/impact use cases, execution and mutation boundaries. Independent implementation packages consume Core data plus explicit configuration/artifact bytes. Installable Modules provide versioned Schemas, Projectors or both; static Go composition implements their capabilities without a dynamic plugin loader.

The prior v0.13 Domain/policy runtime moves to an explicit Host compatibility owner rather than remaining aliases in Core. Its consumers retain their actual behavior and exact regression tests. This isolation is a migration seam, not evidence that their mechanisms need replacement. New Definitions are not lowered into historical Resources as a second canonical model.

## Structural contract to prove first

Definition identity is `(apiVersion, kind, namespace, name)`; package origin and file location are provenance, not identity. Duplicate identities fail even when supplied by different modules. Renames/version changes are remove plus add. Schemas are activated explicitly with exact versions/bytes; there is no inheritance, trait activation or version-range policy precedence.

Each Kind and Property has explicit purpose. Definitions require nonempty purpose. Closed objects reject unknown fields; the compiler does not judge prose quality. Property multiplicity is independent of value type: singleton properties are scalar/object values, many-valued properties are lists even with one item; absent is zero, null is invalid. Counts are nonnegative, ordered, and bounded by finite compilation limits; `unbounded` means within those input limits. Lists preserve order; duplicate reference targets fail. Embedded object typing is closed and depth-limited.

Reference properties name one exact nominal target Kind; omitted API/kind in a reference value use that declared target, never ambient discovery. Namespace is explicit (empty is a real namespace). Kind references resolve exact registered Kind identities and do not activate checks or select every instance implicitly. Cross-Schema references require both Schemas to be explicitly present. Cycles in the canonical graph are permitted unless a higher-layer declared check forbids them; all walks are finite.

Normalized Core edges retain source identity, exact property path, target identity and source provenance. They do not silently activate context/invalidation/acyclic policy. Above-Core consumers declare traversal/assurance meaning. Canonical model digest is deterministic; source revision/evidence remain separately bound. No AI participates in compilation.

## Staged executable migration

1. Commit this target and canonical repository rule, then isolate the historical kernel/consumers and implement a real minimal compiler with source-to-CLI positive/negative proof.
2. Add exact local Module manifests and activation with schema-only Foundation/architecture and projector registrations. Recommend from explicit goals and supplied inventory; recommendation grants no installation/adoption authority.
3. Adapt the validated projection lifecycle to canonical scopes and add separate ProjectionRecord/VerificationResult ownership traceability. Preserve model/snapshot/config/intent/tool/target/protected-path bindings, fresh-plan recomputation, rejection of stale/modified plans and exact reviewed candidate-byte digests. Apply remains materialized-unverified. A Plan is not the durable record: append provenance after materialization and separate evidence after checks. Fixed checks do not themselves prove checker independence or semantic sufficiency.
4. Prove one executable greenfield and bounded brownfield candidate path, two representations, unknown/excluded accounting, independent verification and child-to-parent composition failure. Deterministic/fixed actors prove protocol, not agent quality.
5. Demonstrate intent change versus drift repair and bounded affected-scope review, retaining conservative unknown causes. Preserve prior gauntlet, holdout and adoption negatives.

Each stage must be runnable; interfaces alone are not completion. The full reset remains active until the executable foundations and vertical slice are integrated. Remote marketplace, arbitrary queries, automatic policy adoption, source semantics in Core, universal code generation and continuous distributed operation remain deferred. No stable release is authorized automatically by this source migration.

## Evidence and falsification

Retain architecture dependency gates, strict legacy acceptance, explicit write/partial-failure behavior, fixed-snapshot validation and method-bound evidence. Test unknown fields/identities/refs/counts, cross-Schema and Kind references, determinism, full identity changes, no kernel policy/provider dependencies, stale plans/evidence, owner collisions, unknown artifacts, failed composition and explicit escalation. Record any weakening, authoring friction, model/check duplication or broad impact rather than rewriting negative evidence. Synthetic protocol and finite implementation tests do not establish productivity, token savings, full semantic assurance or human acceptance.