# Projection-first reconciliation

Status: accepted owner-directed source design, 2026-10-05. Baseline: `adef79d935399f8ac63ad874dbdeab8d15c418a1`. Published v0.13.0 remains unchanged. The [vision](../vision.md) owns product intent. This note owns the new bounded local projection contract, not empirical benefit claims.

## Decision and authority

Markitect is the desired-state owner for deliberately governed engineering intent. A governed implementation is a representation of that intent; incidental implementation choices are declared adapter freedom, not an independent architecture authority. Human narrative/rationale, declared external requirements, vendor/tool artifacts and exclusions retain explicit roles. No file outside the declared scope is claimed understood or converged.

Every governed change starts with desired state. Intent changes require changing its canonical owner and replanning before materialization. A bug or stale representation can be repaired against unchanged intent. Neither a fake YAML edit nor retrospective rewriting of intent is required. Discovery remains observation → candidate → human review → explicit adoption.

Core remains opaque to source languages and representation formats. Existing Domains already express custom concepts, typed purpose/behavior contracts, ownership and finite invariants. Behavior carried in a property is not executable proof. No Domain operator, IR provider field or external adapter protocol changes are justified.

## Inventory before implementation

| Subsystem | Classification | Resulting role and gap |
|---|---|---|
| Core, semantic IR, Domains/resources | ALREADY FITS | Generic deterministic meaning; custom vocabulary. No source-language understanding or projection generation follows from a Kind name. |
| Policies, packages, exceptions | ALREADY FITS | Explicit finite checks, exact contracts and visible waivers; neither approval authentication nor business correctness. |
| Context | NEEDS REFRAMING | Relevant desired state supplied to a materializer. Declared closure/provenance remains unchanged; no policy traversal becomes a context edge. |
| Impact | PARTIAL / HYBRID | Semantic consequences remain conservative. Explicit contracts map subjects to target review; no inference of undocumented files or false claim of exact affected source. |
| Agent Rules and Markdown | ALREADY FITS within declared outputs | Deterministic projections with source ownership and exact-byte reconciliation. Ordinary narrative remains separately owned. |
| Artifact Coverage / managed artifacts | PARTIAL / HYBRID | Path accounting lacks AI-projected and vendor roles. Accounting alone cannot establish content consistency. Extend explicit supplied owner facts, not filename inference. |
| Hooks and Pipelines | PARTIAL / HYBRID | Bounded specialized source checks; target representations need explicit ownership and materialization. They do not prove hook execution or hosted CI. |
| .NET | ALREADY FITS as evidence | Literal ProjectReference evidence, not C# behavior or runtime dependencies. Apply remains unsupported. |
| GitHub and Azure | ALREADY FITS as captured evidence | Offline metadata checks, no live provider convergence or Apply claim. |
| Adoption / Copy Me | ALREADY FITS | Owner-selected immutable evidence and separate candidates; review creates authority. No scope expansion or automatic adoption. |
| Markitect-first | NEEDS REFRAMING | Already loads fixed intent before both change classes. Clarify representation repair rather than claim an implemented bypass existed. Guidance alone does not intercept all agent edits. |
| Observe / Plan / Apply / Verify | PARTIAL / HYBRID | Exact renderer lifecycle and external command SPI are separate implementations. Compose local declared representations without replacing those public protocols. |
| Project artifact boundary | NEEDS REFRAMING | Core still sees opaque bytes. Product may explicitly designate them as canonical inputs, observed representations or projection targets. |

Independent source audits were performed before this design; they read neither private Konfyra evidence nor actor outcomes. No claim that the paused A/B proves universal drift or benefit follows from this inventory. Process/controller failures are separate from product results.

## Smallest contract and ownership

An independent `internal/modules/projections` capability owns finite, side-effect-free contract validation, target observations, plans and evidence-relative convergence. It consumes Core IR plus explicit config/artifact bytes/evidence. It imports neither Host nor sibling Modules. Host owns acquisition, static renderer composition, check execution and guarded writes. CLI delegates to Host. No dynamic plugin runtime or agent scheduler is introduced.

A versioned local configuration contains exact contracts: stable ID, exact qualified source resource keys, representation label, materializer name/version and mode (`deterministic` or `ai`), exact unique target paths, named Project verification checks, allowed implementation freedom and explicit contract dependencies. Keys are explicit resolved identities, not a query or arbitrary selector language. A later editor may improve authoring without changing semantics.

Paths must be portable exact relative regular-file paths: no globs, parent aliases, hidden `.git` writes, symlink traversal or case aliases. Targets may not overlap canonical resources, Domain sources, Project/config, declared inputs, tooling or vendor ownership. Contract dependency cycles fail. Each target has exactly one contract owner. External provider objects are not supported by this first local contract; existing provider-specific evidence remains independent.

Host supplies desired bytes only from registered deterministic renderers. For the first slice this is `markitect-render`, reusing Markdown/Agent Rules. An AI materializer supplies a separate versioned candidate: exact reviewed plan digest and exact path→UTF-8 content mapping. It cannot change desired state or contract config, invent paths, mutate vendor/external inputs or claim convergence from its own response. Materializer identity strings and digests are supplied/observed identities, not authentication or model provenance.

## One lifecycle, two realizations

`projection --action observe|plan|apply|verify --config <exact-path>` is an explicit source-only command. Existing render/reconcile/check/context/impact semantics stay intact.

Observe inspects only the supplied snapshot and renderer expectations; it writes no repository files and runs no candidate code. Deterministic mismatch/missing target is drift. AI target existence alone is incomplete semantic evidence. Plan records source/model/config/tool identity, exact observed target hashes, owners, target requirements and evidence needs. Plans are deterministic for fixed inputs; whole-model/unknown inputs conservatively invalidate plans. They are work contracts, not owner approvals.

Apply requires explicit `--write`, a non-protected isolated branch for Git checkouts, a provisional working-tree snapshot and a saved exact plan. Host regenerates the plan, rejects changed intent/config/tool/observation and validates all target paths before writes. Deterministic bytes come from registered renderers. AI candidate bytes are separately digest-bound to that plan; they may vary internally within stated freedom. The first implementation performs sequential local writes under the existing exclusive writer lock, not parallel external mutations. Partial failure reports exact written paths; no rollback, deletion or automatic retry is implied.

Verify runs explicitly named Project checks against an immutable candidate snapshot using the existing fixed-snapshot verifier. Check evidence must bind that snapshot, desired contract and exact target bytes. Evidence is not accepted from an arbitrary user-authored pass record. Missing/unavailable check is incomplete; failing checks or deterministic mismatch are drift. Policy noncompliance blocks materialization/acceptance; existing read-only failed-policy Context/Impact remains usable. Waivers remain visible as governance deviations, never silently relabeled policy passed.

Converged means: structurally sound policy-governed model, required selected targets present, deterministic targets exact, required bounded checks passed over the fixed observed snapshot, and configured artifact accounting satisfied. It means no contradiction detected by those declared checks, not complete program correctness. Files outside configured coverage are unassessed. Verify evidence reflects an exact snapshot, not continuing runtime truth.

## Project local-projection registration

The local projection contract is selected explicitly by one Project `spec.adapters` entry with `type: local-projection`, `version: v1alpha1`, and only the closed config keys `contracts` and `coverage`. Both values are exact portable project-relative file paths; globs, aliases, traversal, reserved paths and path overlap are rejected. Paths are opaque files to the registration boundary and need no filename convention. At most one such registration is allowed. `contracts` names the projection contract configuration, while `coverage` names the required exact-path artifact coverage declaration; missing coverage is incomplete/invalid configuration, never implicit whole-project coverage. These paths are adopter-owned inputs and participate in ordinary fixed-snapshot Verify, Context and Impact input binding.

The registration selects Host orchestration only. It does not add authoring semantics to Core, import a Module into authoring, or replace the existing command-adapter protocol. Coverage is required for planning and before Verify can claim configured convergence. Every AI target requires a separate candidate file plus the exact expected raw SHA-256 supplied by the invoking Host workflow; absent or partial candidates are rejected rather than being inferred from previous output. Deterministic-only Apply continues to use the registered renderer without AI candidate evidence.

Context remains the relevant desired semantic closure. Impact remains semantic change causes plus a conservative review set; contracts expose the associated target fan-out with explicit owners. Unknown impact remains conservative. No generated output becomes a semantic owner and adapters never reconcile against one another's output as truth.

## Proof and limits

An executable greenfield example must define a previously unknown Kind, purpose, typed behavior, explicit relation/constraint, process and projection contracts. It materializes code/tests/CI/hooks through fresh AI candidates and docs/provider instructions through existing deterministic renderers. Independent project-owned checks assert finite behavior and literal configured process expectations. AI-authored tests alone are insufficient proof.

Negative cases corrupt code, docs, Codex, Claude, CI and hooks separately with unchanged desired state; observe/verify must report drift/incomplete and explicit repair must converge. Intent evolution fans out to declared representations. Two fresh isolated AI candidates may use different private names/algorithms while passing the same predeclared verification; this proves bounded allowable variation, not productivity or obedience outside coverage.

Historical A/B records stay frozen/paused under PR #78, no overall winner, unopened holdout. Private Konfyra scope/adoption authority and published immutable assets are untouched. Release readiness is separate. Benefit and maintenance cost need a later real-adopter comparison.

## Reviewed candidate and independent-check boundary

AI Apply requires the complete declared AI target set and an explicitly supplied `--expect sha256:<hex>` matching the exact external candidate record bytes. The parsed object is also bound to those bytes; changing file contents while retaining a plan digest is rejected. Deterministic-only configurations need no AI candidate. Successful Apply reports only exact writes and materialized-unverified state, not acceptance. The Host writer workspace is reserved and cannot be a target.

Projection verification runs each declared check in a freshly materialized copy of the same immutable snapshot. A command that changes, removes, or replaces an original snapshot file invalidates its evidence. Later checks never inherit a previous check's changes. Extra temporary check outputs are not adopted as observed state. This is bounded snapshot-integrity detection, not OS isolation or prevention of writes outside the temporary tree. Existing ordinary unregistered repository verification retains its contract. Checks remain explicitly trusted local executions; tool strings, model/materializer identities and successful exit codes do not authenticate semantic truth.


## Opaque artifact and Domain provenance boundary

An active registration reserves its exact mapping/configuration files and target paths as opaque artifacts, including CI/provider YAML inside an Area. Files declaring an active canonical API are rejected at those paths; Apply also rejects such envelopes before writing. Coverage's own noncanonical API envelope remains an opaque module configuration. Registration cannot hide Project, Domain, or typed resource authority.

Active Domain descriptors are explicit projection sources using `domain:<apiVersion>/<name>`, with exact source/package/version provenance from the same normalized model. Context already supplies active Domain inputs; their contracts are therefore included without adding policy or context edges. Impact includes a Domain-only contract when its source, pinned archive, Project activation or contract configuration changes. Unrelated narrative edits do not imply that Domain fan-out.

Fresh-check integrity compares fixed bytes, regular-file identity and, on supported Unix platforms, executable mode. Windows file permissions do not encode the Unix executable bit and no equivalent assurance is claimed there. This remains post-execution bounded detection, not a sandbox.

Impact also closes over explicitly declared contract prerequisites and records `viaContracts` for dependent review targets. This is bounded contract status/review propagation; Context exposes `dependsOn` without automatically including prerequisite sources or creating semantic/context edges.


Generic verification evaluates the owner's configured check contract; it cannot prove checker independence or semantic sufficiency. In particular, an AI-owned test may be selected as the sole check. Such a configuration gives weak self-produced evidence even if it meets its configured contract. Verify output names this boundary. Independent checker ownership is established separately by the synthetic proof, not inferred from argv or promised universally. A later explicit evidence-input/ownership design must address wrappers and trust before any stronger general assurance claim.

Host Plan composes the pure contract plan with artifact accounting and required native-renderer findings, then binds those sorted causes into the plan digest. Matching per-contract targets do not imply overall convergence. Plan and Observe remain read-only and report missing runtime check evidence as incomplete; fixed Verify alone may establish configured convergence. This does not add a policy assertion or weaken Apply freshness.

Verify uses the same composed aggregate status in its nested plan, retaining matching per-contract states separately. Accounting/rendering and required-check causes participate in the final plan digest; a failed required check outside a contract’s selected evidence cannot leave the aggregate plan converged.
