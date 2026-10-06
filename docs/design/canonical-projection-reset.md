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

Host owns source codecs, Module manifest/catalog activation, explicit source selection, compatibility lowering, context/impact use cases, execution and mutation boundaries. Independent implementation packages consume Core data plus explicit configuration/artifact bytes. Every installable Module has exactly one type: `schema` or `projection`. Schema Modules provide canonical language only; Projection Modules supply target expertise and execution tools only. Mixed packages and Schema-to-Projection dependencies fail activation; static Go composition implements their capabilities without a dynamic plugin loader.

The prior v0.13 Domain/policy runtime moves to an explicit Host compatibility owner rather than remaining aliases in Core. Its consumers retain their actual behavior and exact regression tests. This isolation is a migration seam, not evidence that their mechanisms need replacement. New Definitions are not lowered into historical Resources as a second canonical model.

## Explicit projection binding

Installation registers capabilities only and causes no materialization. A Foundation `Projection` declares a desired representation type, exact selected canonical scope, target and policy identities. Project configuration separately binds that requirement to an installed Projection Module. `ProjectionPolicy` constrains representation decisions; it is neither a generator template nor the declaration that a representation should exist. `ProjectionRecord` identifies that Projection and the actual materialization separately.

The Host constructs a bounded request containing explicitly selected Definitions, relevant Schema contracts, resolved references, selected policies and supplied target state. Whole-model/repository context is not implicit. Definitions and artifacts have many-to-many ownership. Insufficient representation guidance is an escalation rather than an invented mapping. Nominal typed References cannot express arbitrary ontology members through one Foundation property: the first source scope uses closed full identity objects resolved by Host. No polymorphic-reference or traversal primitive is introduced for that authoring tradeoff.

## Strict Module types and unified execution

The latest owner addendum replaces the earlier mixed-capability direction. Every installable Module is exactly a Schema Module or a Projection Module. Schema Modules provide at least one Schema and no execution capability; Projection Modules provide target execution capability and no Schema. Bundles are installation convenience, not a third type. Technologies needing both use separately pinned packages. Go application/infrastructure helpers are not automatically installable Modules.

All projection work follows one bounded Executor -> candidate -> Verifier process. Markdown renderers, compilers, candidate ingesters and architecture checks are tools in that process, rather than a separate top-level deterministic projection architecture. Existing byte-strategy differences remain useful internal Plan/evidence mechanics and retain their historical release meaning. Materialization never constitutes semantic verification; fixed checks cannot authenticate independence or establish complete semantic sufficiency.

The latest reconcile-resolution addendum supersedes canonical `spec.projectionModule`. Foundation now selects `spec.representation` as the desired target type. A separate `projectionBindings` entry in source/project configuration selects the exact installed Module name. It introduces no public Projector Kind or canonical execution mechanism. Host resolves that Module's unique registered execution entrypoint; multiple entrypoints are explicit ambiguity instead of hidden first-match precedence. Internal entrypoint ID/version remain useful operational provenance and static composition identifiers. Future multi-entrypoint cases need evidence before extending canonical selection.

ProjectionPolicy is project-owned representation intent. Generic symbolic Markdown can faithfully present selected purposes and values without a per-Kind technology mapping. The first .NET candidate tool requires explicit per-Kind guidance as its conservative input contract. The generic binder does not judge prose adequacy. Neither guidance presence nor candidate byte-shape acceptance proves sufficient semantics; the agent must escalate missing, ambiguous or conflicting intent. Target tools gain no hidden ontology mappings.

Pressure tests distinguish protocol feasibility from AI reasoning quality: supplied candidates for custom Kinds demonstrate no hard-coded ontology lookup but do not establish that AI interprets arbitrary third-party semantics correctly. Independent agent judgment, its cost and sufficient assurance remain hypotheses requiring a separate bounded experiment. Prior negative evidence remains intact. The exact owner addenda are archived with the earlier inputs.

## Reconcile-first resolution

The current operating principle is **Users change canonical intent. Markitect reconciles representations.** The high-level planning path compares fixed canonical snapshots, follows explicitly resolved reference dependencies, identifies configured desired representations, incorporates caller-selected active records and target drift, and returns bounded work scopes. Targeted request/plan/apply operations remain Executor tools and diagnostic surfaces, not the normal request to implement individual files. No background controller or implicit write is introduced.

Canonical Projection remains useful as a reviewed statement that a representation should exist. Its desired type, semantic scope, location and selected policies survive replacement of a compatible execution package. Removing it would move that existence decision into capability-enabling configuration and lose the explicit distinction demonstrated by installation-without-materialization. Runtime `projectionBindings` owns the concrete Module selection, exact pins stay in package configuration, and records retain the actual selected mechanism. No match, duplicate binding, incompatible target or ambiguous capability fails explicitly; no hidden precedence resolves it.

The first high-level reconcile plan can derive affected representation requests without an explicit implementation command. It remains read-only and does not itself run an autonomous model, authorize Apply or claim convergence. Reused guarded execution and immutable verification provide the bounded steps for agents. Future agent-led scheduling must preserve this plan contract and explicit authority. Scope-directed work and conservative plan/evidence invalidation are distinct: global model/revision bindings can still invalidate unrelated plans; the output must disclose that limitation rather than calling it precise local freshness.

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