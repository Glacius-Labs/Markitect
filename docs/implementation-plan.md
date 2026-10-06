# Markitect roadmap

Updated 2026-10-06. This is the canonical owner of shipped source and planned-work status. v0.13.0 remains the latest published release, adding read-only policy-failure analysis, bounded selective handoff, .NET/source hardening, the provider-neutral Markitect-first workflow and managed-artifact accounting. v0.12.0 adds the bounded `same-target` assertion to reusable architecture contracts; per-subject policy outcomes, explicit source-bound exceptions, and the Copy Me authoring workflow shipped in v0.11.0. The [production assessment](production-assessment.md) records release verification and asset digests. v0.10.0 remains the historical baseline that shipped the generic canonical engineering model, normalized semantic IR, bounded constraints, and configured adapters. The prior v0.9.1 release corrects source-relative prose navigation and retains the v0.9.0 separation of optional Markdown views, direct provider outputs and consumer tool pins. Dated v0.11.0 release evidence and the v0.10.0 historical baseline are in the [production assessment](production-assessment.md), and available distributions are listed in [GitHub Releases](https://github.com/Glacius-Labs/Markitect/releases). [Architecture](architecture.md) describes current behavior and the source-only canonical reset direction, [Usage](usage.md) records supported syntax and CLI behavior, the [canonical engineering plan](canonical-engineering-plan.md) owns model decisions, and [Operations](operations.md) owns release gates.

## Canonical reset and projection execution (source-only alpha)

The [accepted reset design](design/canonical-projection-reset.md) supersedes the earlier projection-first source architecture while preserving its validated reconciliation behavior and trust boundaries. v0.13.0 remains the latest published contract; the reset is source-only alpha, not a release claim. The [vision](vision.md) owns product intent and human/agent responsibilities.

The candidate introduces a minimal pure Core structural compiler for Schema, Kind, Property and Definition. Historical v0.13.0 policy/kernel consumers keep their exact behavior behind Host compatibility and are not obsolete. Installable Modules are strictly Schema-only or Projection-only; Go capability packages remain separate implementation units. Host owns explicit source selection, manifest activation, static entrypoint composition, safe persistence and execution.

The first end-to-end slice uses locally versioned Schema Modules and Projection Modules. A desired Projection names representation semantics and exact scope; source configuration binds that intent to one exact installed capability. Host resolves version pins and a unique internal entrypoint. Changing a compatible binding must leave canonical Definitions/model digest unchanged while staling bound plans and records. Installing a Module alone creates no canonical Projection or materialization.

The owner-preferred high-level interaction is change intent → aggregate `plan` → reviewed `apply`; introduce these commands only with a coherent aggregate executable plan, not as aliases for narrow candidate tools. The conceptual execution is one bounded Executor Agent → candidate representation → Verifier Agent with independent evidence. Deterministic renderers and other tools operate inside that lifecycle. Current alpha behavior is narrower: Host derives bounded requests from canonical impact, active records and observed targets through `reconcile-plan`; targeted plan/apply remains a candidate tool flow. Independent Projection Module proposals and autonomous recursive orchestration are future work. Unknown scope/ownership broadens review; conflicts stop before execution. Operational ProjectionRecords and VerificationResults remain separate facts, not canonical intent or proof of authorization, semantic adequacy or acceptance.

Brownfield adoption is an explicit, read-only stage before normal reconciliation: inventory and inference propose candidate intent; an owner reviews/corrects and accepts canonical intent; exact existing artifacts are matched to that scope; immutable checks verify the source and target; only valid existing bytes can be recorded with `origin: adopted`. The source alpha `adopt-plan`/`adopt` path now supports explicit exact-path selection, caller-supplied active ownership claims and an owner-supplied review reference. It generates, rewrites and persists nothing, selects no active record automatically, and does not authenticate semantic review. Unmatched target artifacts are UNKNOWN and normal reconciliation escalates them. Focused Host adoption and records checks pass, including zero-churn preservation and failed/stale verification; final quality status remains commit-bound to the [validation report](validation/canonical-projection-reset.md) and integration PR receipts.

The [alpha CLI/API](canonical-projections.md) exposes structural model/context, explicit request, impact, Host-derived read-only reconcile planning, bounded candidate plan/apply, and immutable verification. Execution remains staged: (1) isolate and compile the new Core while preserving legacy semantics; (2) enforce strict Schema/Projection Module types; (3) complete greenfield materialization and brownfield zero-churn adoption with fixed evidence; (4) pressure-test custom Kinds, unknown/excluded inventory, stale input/plan/evidence and parent composition failures. Each stage needs runnable positive and negative evidence. Synthetic protocol tests do not prove AI reasoning quality, human-attention savings, complete assurance, adopting-project acceptance or release readiness. Release remains a separate decision after exact-head gates and review.

## Capability proof checkpoint (source validation)

The [proof program](research/proof-program.md) freezes architecture and inventories implemented capabilities before new experiments. The initial inventory found the explicit full-acquisition stop condition: the canonical reconcile path loads complete base, candidate and working snapshots before deriving scoped work. Its [reproducible witness](../experiments/capability-proof/README.md) distinguishes local work selection from global reads and conservative evidence refresh. The [matrix](research/capability-matrix.md) retains narrow existing support without promoting it to new proof.

Downstream AI, goal-led greenfield, actual owner-reviewed brownfield, human-attention A/B, concurrency, longitudinal and technology re-projection trials are NOT RUN at this gate. Assess the acquisition/target-observation/freshness boundary before further execution. This does not weaken unknown-artifact visibility, snapshot digests or write guards, add a capability, complete the proof phase, or publish a release.

## Clean architecture consolidation (current source)

The [consolidation decision](design/clean-architecture-consolidation.md) records the historical post-v0.13.0 package isolation. The [canonical reset](design/canonical-projection-reset.md) now stages a new minimal Core beside a Host compatibility owner for the historical kernel and consumers. The prior projection implementation remains the validated lifecycle source; the reset does not add a parallel projection engine or alter the published release. Go imports are checked across production, tests and platform files. A Module consumes Core values and its own configuration/explicit artifacts; sibling and Host imports are forbidden. Public Domain operators, schemas and adapter protocols do not expand.

Git Hooks and Pipelines have bounded source-only checks, invoked through `cmd/markitect-check-modules` and the root Project. The historical agent-rules, Markdown, artifact accounting, Git hooks, pipelines, .NET and offline GitHub/Azure consumers retain their cohesive subtrees under Host compatibility. New Markdown/.NET capability packages consume the new Core; selective capture/review remains an independent implementation helper. Content-package authoring remains with Host, source acquisition with Infrastructure, and immutable distribution/bootstrap with Tooling. The [validation report](validation/clean-architecture-consolidation.md) records compatibility, negative patches, Core measurements, dogfood and exact-source gates. Source structure does not update the published v0.13.0 release.

Future capability work should default to independent Module changes plus explicit Host wiring. Shared semantic requests remain coordinator-owned and need concrete generic evidence. Reconcile-first projection resolution is the active source direction under the accepted design above; it does not authorize reading or adopting any private consumer's candidate semantics. A coherent source candidate must pass Windows/Linux and immutable package/bootstrap gates before a separate release decision; this roadmap update does not publish automatically.

## Product priority and evaluation

The canonical [vision](vision.md) defines the goal: human-owned engineering intent supporting autonomous work within deliberate boundaries. Prioritize a concrete routine human activity that can be replaced or made unnecessary, with explicit authority, evidence coverage and measured total upkeep. Feature count, generated text, more provider surfaces and raw agent throughput alone are not evidence of progress. [Measurement](measurement.md#human-attention-and-delegated-work) owns the future evaluation criteria.

The matched comparison remains inconclusive and the existing mature consumer already has canonical owners and generated pointers. No new benefit claim follows from stronger positioning. The selective-capture/storage/handoff source slice below makes a bounded owner-led candidate review possible; a repeated task comparison should then measure interventions, review and maintenance against simpler guidance plus architecture tests. Those remain validation priorities, not automatic adoption or benefit conclusions. The [workstream map](workstreams/README.md#connection-to-the-product-vision) records each mechanism's intended contribution and current proof limit.

Continuous agent operation and reduced per-change human review are aspirational consumer workflows. A scheduler/background operator, automatic acceptance, live provider Apply, additional interfaces and new Core primitives remain separate evidence-led decisions. The [alignment assessment](design/product-vision-alignment.md) records this gap without changing current trust boundaries or rewriting historical pilot findings.

<a id="selective-adoption-handoff-unreleased-source"></a>
## Selective adoption handoff (v0.13.0)

The [focused contract](design/selective-adoption-handoff.md) closes the first wave's preparation/Copy Me integration dependency. `prepare` opens exact owner-selected Git blobs at full commits, previews a reproducible handoff, and exclusively stores selected evidence in an external workspace only with the reviewed digest. `copy-me` validates explicit evidence/candidate/decision records against that handoff without source acquisition, model calls, authentication or canonical adoption. Existing greenfield Init and generic snapshot/Domain semantics remain unchanged. Optional ContextRun evidence is supported but not required.

The [validation record](validation/selective-adoption-handoff.md) covers acquisition-spy, alias/refusal, partial-storage, staleness and end-to-end CLI controls. They establish narrow behavior, not discovery quality. The earlier public sanitized Konfyra replay remains distinct: a separate, restricted owner-approved capture covered 36 selected files and four bounded interpretation candidates outside both repositories. Public documentation contains no private identifiers, paths, digests or raw content. This does not establish candidate acceptance, canonical adoption or discovery quality. The next product validation is an owner-reviewed real-task comparison including initial setup and recurring maintenance; automatic discovery/adoption, source inference and Core expansion remain outside this slice.

<a id="first-parallel-development-wave-unreleased-source"></a>
## First parallel development wave: shipped and source-only boundaries

The [wave record](validation/parallel-development-wave-1.md) tracks ten bounded assignments from the verified preparation baseline. .NET literal XML namespace hardening and the applicable source controls are included in v0.13.0. Independent offline GitHub/Azure repository-metadata consumers remain experimental source-only interfaces using the existing command/model protocol; captured records establish correspondence to fixed inputs, not live provider state. No live provider Apply is supported.

Init and Copy Me contributed [preparation](design/init-adoption-preparation.md) and [evidence-review](design/copy-me-evidence-review.md) proposals. That wave deferred durable implementation; the separate selective handoff slice above resolves its shared contract. Greenfield Init remains unchanged. The sanitized consumer inventory establishes scoped observations, not migration, complete coverage or benefit. The cross-adapter runner control uses actual compiler/runner DTOs. MCP remains experimental. Historical release status is recorded below.

<a id="read-only-policy-failure-analysis-unreleased-source"></a>
## Read-only policy-failure analysis (v0.13.0)

The [parallel-development preparation](development/README.md) consolidates invariants, classifies risks and records current adapter/adoption seams. Its [workstream map](workstreams/README.md) prepares future assignments without implementing init/discovery expansion, adapters, Konfyra migration or another Core primitive. External command adapters support independent read-only work; shared native rendering and cross-command external-target ownership remain explicit coordination points. The exact verified integration commit becomes the dispatch baseline, separately from the published release.

The real-code pilot exposed a concrete architecture-migration workflow gap: ordinary failed PolicyResults blocked Context and Impact before a reviewer could inspect the candidate. The [focused design](design/policy-failure-analysis.md) reuses the existing resolved graph and normalized model, distinguishing structural invalidity from policy noncompliance. Explicit `--analyze-policy-failures` on resource Context and fixed Impact produces visibly marked read-only analysis; failed policy still exits 1. Strict defaults, Check, Verify, exceptions and reconciliation remain unchanged. No Domain operator or broader graph capability is added.

Impact reports direct PolicyResult changes/subjects separately from its conservative affected set and causes. Context retains declared closure and file inputs. A separate [MyMeetings replay](../experiments/policy-failure-analysis/README.md) uses the prior pilot's exact preserved revisions without editing historical evidence. This measures inspectability, not productivity, token savings or defect reduction.

The [matched three-task comparison](validation/agents-md-vs-markitect.md) against a short AGENTS.md plus existing architecture tests is complete. Its overall result is inconclusive: simple implementation candidates work; Markitect explains noncompliant contract evolution, but its additional maintenance cost is not shown to pay for itself. No independent truth owner or concrete human review step was removed. The next validation step is a repeated adopter study with matched policy history, consistent permissions/SDK observation and stronger instrumentation. Missing ADR/source selection, authoring friction, duplicated ownership mapping and broad context inputs remain separate follow-ups. These experiments do not repair the historical negative findings or justify another Core primitive. Those findings remain unchanged by v0.13.0.

## v0.12.0 — bounded Domain equality (published)

Starting from published v0.11.0, this iteration tests a reusable modular DDD / Vertical Slice [Software Architecture package and executable consumer](../examples/software-architecture/README.md). It exercises structural failure cases, policy selection, exact package v1 → v2 adoption, affected subjects, a narrow temporary exception and its removal after updating the declared architecture. Application code is not migrated.

The [language-pressure report](design/domain-language-pressure.md) and [experiment notes](design/software-architecture-experiment.md) own the enforced-versus-unexpressed rule matrix, counterexamples, design conclusions and evidence limits. The follow-up [resolved-target equality assessment](design/resolved-target-equality.md) first proves the same ownership comparison is missing in Software and an independent [Delivery topology](../examples/delivery-target-equality/README.md), then adds only `same-target`: two named relation paths of at most two singleton steps, compared by canonical GraphKey. It exposes result traces, digest-bound traversed inputs and derived policy dependencies without creating Context edges. New exact Software package versions 1.1.0 and 2.1.0 enforce a selected Feature-ownership cohort while preserving historical 1.0.0 and 2.0.0. Aggregate fan-out, inverse ownership, module-pair allowlists, mixed-relation cycles and implementation semantics remain explicit gaps. No query DSL, Pattern, Trait, inheritance or implicit Domain composition is introduced.

Small source improvements expose Domain identity in source provenance, relation causes in context and impact, and bounded impact for per-resource policy edits when relation effects permit it. Selection-wide constraints, inventory/configuration/unknown changes and legacy policy contexts lacking derived invalidation retain conservative impact; same-target result consumers use their explicit Context closure. Windows/Linux CI and packaged-command gates include the new fixture. The immutable v0.12.0 release is verified. Its synthetic architecture and Delivery fixtures demonstrate the published behavior; they do not establish real-project adoption or business benefit.

## v0.11.0 — reusable architecture contracts (published)

v0.11.0 extends the v0.10.0 generic Domain model so adopting Projects can use versioned package Domains as ongoing architecture contracts. It adds scoped count semantics, structured per-constraint/per-subject PolicyResults, narrowly scoped explicit policy exceptions, and a provider-independent Copy Me workflow that prepares human-reviewed candidates from explicitly selected fixed evidence. The [engineering constitution](engineering-constitution.md) owns the adopted product target; the archived follow-up [strategy sources](strategy/README.md) preserve the original proposals and their non-authoritative status.

**Shipped contract:**

- Architecture concepts use existing project-owned or package-supplied Domains. Constraints for a shared resource shape compose in one Domain for the active API version; a `conformsTo` relation does not activate another Domain's constraints. The core has no `Pattern` kind or separate pattern DSL.
- `count` supports `scope: resource|selection`: per-subject relation bounds versus a count over the selected population. Selection membership changes, including additions/removals, affect results.
- `PolicyResult` records expose `passed`, `failed`, or `waived` status through the normalized model, `check`/`verify`, and relevant context output. Context includes results for closure subjects plus collection results for global selection constraints.
- Projects support at most 64 `policyExceptions`, each bound to its exact `apiVersion`, constraint name and digest, subject GraphKey and digest, rationale, owner, and recorded decision. Constraint digests also bind referenced relation semantics. Optional `expiresOn` requires an explicitly frozen `policyDate`.
- `policyDate` is a fixed as-of value included in snapshot evidence. An exception expires when `policyDate >= expiresOn`; waived PolicyResults retain the original violation and expose the exception rationale, owner, decision, expiry, and policy date.
- Exceptions apply only to per-resource constraints. Type/schema errors, reference resolution, cycles, selection-wide checks, relation bounds, and other global structure remain unwaivable. Malformed, unknown, stale, unused, and expired exceptions fail checks; exception metadata is not authenticated approval.
- Exact pinned package updates are the architecture-contract migration path. A package update changes checks and impact evidence; Markitect reports affected subjects and results, while the adopter reviews and changes implementation code separately.
- `Copy Me` is a provider-independent authoring workflow over explicitly selected files at a fixed snapshot with recorded hashes. It guides an authoring agent to surface observations, hypotheses, counterexamples, and uncertainty in a separate staging candidate; explicit human adoption makes it canonical. The deterministic core has no discover CLI or model calls.
- An explicit `markdown` target generates normalized Domain schemas, relations, and constraints under `docs/markitect/_domains/*.domain.md`, plus relevant per-resource PolicyResults and visible waiver decisions. Generated views remain derived.
- The kernel constraint language remains finite and deterministic.

CUE and OPA remain future specialist-adapter options only if repeated concrete architecture policies exceed the readable kernel assertions. Compare [CUE validation and incomplete data](https://cuelang.org/docs/concept/how-cue-enables-data-validation/), [OPA Rego](https://www.openpolicyagent.org/docs/policy-language), and [OPA external data](https://www.openpolicyagent.org/docs/external-data) against those same cases before adopting an engine. This is an expressiveness trigger, not a shipped integration or a performance/market claim.

**Release proof:**

The [engineering constitution fixture](../examples/engineering-constitution/README.md) exercises a UseCase policy with exactly one required Handler and explicitly described optional extension points. Its versioned package Domain supplies the structured policy used by generated human-readable Domain/resource views, bounded agent context, and deterministic checks. The fixture covers an exact package-pin update, `impact`, changed per-subject PolicyResults, an explicit source-bound waiver and its removal after implementation, without automatic source-code rewriting.

The [engineering discovery fixture](../examples/engineering-discovery/README.md) replays a synthetic one-repository Copy Me workflow using an exact fixed snapshot and selected-file hashes. It records observations, a supporting item, a counterexample, and a separate candidate and decision dossier. Its deterministic helper checks byte bindings; it does not authenticate a reviewer or adopt policy. These fixtures demonstrate bounded scenarios, not broad semantic correctness, human acceptance, measured consumer effort, or real-world business benefit. Code-level and file-presence claims still require explicit configured checks/adapters; the generic core does not prove that Queries are non-mutating or that Handler/Test/Docs files exist.

The v0.10.0 [canonical engineering example](../examples/canonical-engineering/README.md) remains the historical consumer proof for that release. Compatibility with v0.9 projects was not a design constraint for v0.11.0.

## v0.10.0 — canonical engineering model (published)

This version makes domain vocabulary data-driven and gives its consumers one normalized, deterministic model. Descriptive architecture and normative policy can refer to the same typed resources; configured consumers use validated semantic data rather than interpreting authoring YAML independently.

**Shipped contract:**

- Added a versioned Domain contract loaded from the fixed Project snapshot before resource instances. Domain definitions provide closed, typed resource descriptors, relation descriptors, and a bounded constraint language.
- Refactored resource instances to a generic data shape while preserving qualified identity and source provenance. The supplied AI-working Domain expresses `Rule`, `Workflow`, `Skill`, `Agent`, and related vocabulary through the same kernel.
- Built a normalized semantic IR after Domain validation and reference resolution. Core checks, context, impact, projections, and adapter planning consume this representation.
- Gave each relation explicit context, invalidation, and cycle behavior. Bounded constraints over selected resource sets respond to additions and removals as well as edits.
- Configured adapters explicitly from versioned mappings, including local projections and the external command protocol through observe, plan, apply, and verify, with observation digests bound into results.
- Added an executable, bounded policy slice where the same declared values appear in human documentation, agent context, and a deterministic check, with source and model identity visible across all three consumers.
- Updated CLI, schema output, authoring resources, format/reference documentation, operations, and release material for the shipped contract.

The executable consumer proof in [canonical-engineering](../examples/canonical-engineering/README.md) defines Software and Delivery Domains without adding kinds to the executable. It demonstrates a single policy value flowing into generated resource documentation and agent context while changing deterministic check results; separately configures relation context/invalidation; exercises missing, wrong-kind, and cycle diagnostics; and observes external drift without a canonical model change. The .NET reference adapter adds a read-only check of mapped literal project references against canonical `dependsOn` relationships; unsupported MSBuild semantics remain incomplete. Results bind to their source, Domain, Project configuration, mappings, adapter identity, and observation inputs. Human review, semantic truth, and authorization remain distinct from successful machine checks or apply operations. This is one bounded consumer proof, not evidence of general semantic correctness, production adoption, or business benefit.

This version does not promise arbitrary executable plugins, a general-purpose query language, source-language parsing in the kernel, background reconciliation, or model-based validation. The kernel and external command boundary remain deterministic and explicit. Compatibility with v0.9 projects is not a v0.10.0 design constraint; adoption and migration guidance must explain the new model without retaining legacy semantics as hidden core policy.

## v0.1.0 — historical release

The initial release established a standalone Go/YAML tool, typed resources, explicit dependencies, fixed Git snapshots, context and impact queries, core authoring, generic rendering, and verified private release assets. Its product-source checks and release evidence are summarized in [the v0.1.0 assessment](production-assessment.md). The old Project profile and adopter-specific migration/rendering compatibility must not be carried forward as general product policy.

## v0.2.0 — standalone verification and rendering

The v0.2.0 model makes repository behavior explicit and removes adopter-specific assumptions.

1. **Explicit Project checks.** Remove `Project.spec.profile`. Add optional `Project.spec.checks` entries containing a stable name and `run` argv. The first argv element is a bare executable resolved from `PATH`; arguments are literal and no shell is inserted. `verify` runs the commands from its fixed snapshot, with existing time/output bounds. No declared checks means `incomplete-evidence`, not success.
2. **Explicit outputs.** Keep generic managed views. Render other outputs only through Project-declared `targets` and `ruleAdapters`. Do not load a compatibility renderer implicitly.
3. **Remove migration policy from core.** Remove the built-in migration command and hard-coded conversion adapters. A project that needs to import existing content owns a script and its validation in its own repository.
4. **Keep authoring in core.** Continue shipping portable authoring resources and structural queries with Markitect. They explain the product's resource model without deciding task semantics or requiring a model API.
5. **Prove the neutral boundary.** Test schemas, checks, rendering, fixed snapshots, and the executable example in standalone Windows and Linux CI. Ensure there is no bootstrap, Python, or adopting-project gate inferred from files or repository identity.
6. **Document the v0.1 transition.** Describe removal of the Project profile and explicit checks/outputs using neutral examples. Treat v0.1.0 as an immutable historical release; verify the exact source commit, gates, and assets before publication. The confirmed v0.2.0 release evidence is in the [production assessment](production-assessment.md).

Acceptance: a project can author and structurally check resources without repository-specific setup; `verify` reports incomplete evidence when checks are absent and executes only declared argv on the selected fixed commit; generic views render without a provider; extra outputs require explicit configuration; core authoring still compiles; all release gates identify one source commit and platform.

## v0.3.0 — direct offline content packages

This source version added exact direct package pins in `Project.spec.packages`, deterministic archives from fixed Git revisions, explicit exports, package-qualified graph identities, and package inputs in compiled context and impact/review eligibility. No second content lock or package resolver was introduced; `markitect.lock.yaml` remains the CLI distribution lock. Imported Rules are not activated implicitly, package resources remain read-only, and package manifests do not declare checks, targets, or rule adapters. See [Content packages](content-packages.md) for the contract.

## v0.3.1 — binding-cycle correction

This correction rejects binding cycles that include a Contract selected by its own implementation. The dated source and release evidence is in the [production assessment](production-assessment.md).

## v0.4.0 — minimal project initialization

The v0.4.0 release adds `markitect init` for an existing repository. Preview prints the exact minimal Project YAML and area README plan without writing, including outside Git. `--write` recomputes and validates the plan, requires a named non-protected Git branch, and creates only `markitect.yaml` and one README in a previously absent area directory. It does not infer project policy, add resources or checks, select outputs or packages, use custom templates, or edit root documentation or agent instructions. File creation is exclusive and guarded by the shared write lock; a partial failure reports created paths and recovery guidance, because two file creations are not a transaction.

Acceptance is structural: the new Project parses, resolves, and passes output checks. `verify` must remain incomplete until the adopting project owner declares actual checks and commits the candidate. Consult GitHub Releases for currently available distributions.

This source version also adds an executable project-artifact input example using a Go file, fixes shared-writer path handling for Windows short/long names while retaining reparse rejection, keeps benchmark fixtures checked in CI, and bundles third-party notices accessible offline through `licenses`. The example exercises explicit context and impact, not source-code analysis; the [architecture boundary](architecture.md#project-artifact-boundary) applies to all artifact types. The [authoring pilot](authoring-pilot.md) records actual measurements and the incomplete model comparison; it makes no savings claim.

## v0.4.1 — licensing and repository presentation

The published v0.4.1 release adds the Apache License 2.0 to the product repository and includes it in source distributions. Earlier immutable tags and their attached archives do not gain a license file retroactively. It also improves the README entry, support and security guidance, and contribution templates. Its product behavior and verification boundaries remain those of v0.4.0. The reviewed source, Windows/Linux gates, exact tag, and immutable release verification are recorded in the [production assessment](production-assessment.md).

## v0.5.0 — explicit adapters, measurements, and assertions

The published v0.5.0 release adds project-configured Codex, Claude, and shared entrypoints with strict inventory; see [Provider adapters](provider-adapters.md). It adds a post-publication Windows/Linux benchmark workflow that measures both the new and prior immutable release on fixture v1, with raw results and a summary as workflow artifacts; see [Measurement](measurement.md). The opt-in [consistency MVP](consistency.md) compares explicit, quoted functional assertions and reports source and owner evidence. It does not interpret free text or change human acceptance.

The tagged source passed Windows/Linux release gates, and the immutable release and four attached assets were verified. The release-triggered benchmark failed during runner setup because an action pin was truncated; the separately dispatched repaired run completed on Windows and Linux with raw measurement artifacts. Those measurements are diagnostic, not an acceptance gate or evidence of a speed gain. Consumer pins, project-specific adapter ownership, candidate verification and human acceptance belong to adopting repositories.

## v0.6.0 — documentation placement and optional routers

The published v0.6.0 release adds portable authoring guidance for locating an existing canonical documentation owner and an opt-in `Project.spec.documentation.roots` router check. An adopting project's documentation taxonomy, artifact-to-document relationships, and provider-adapter gates remain project-owned. [Documentation routers](documentation-routers.md) owns the exact participation, link-normalization and diagnostic contract.

Implementation sequence: (1) define and test participating directories and local link normalization against snapshot paths; (2) add strict Project configuration and regenerate its schema; (3) surface stable router diagnostics through `check` and fixed-revision `verify`; (4) update embedded authoring guidance and validate a placement task where an agent finds and changes an existing canonical source; (5) run source and executable-example gates. The acceptance case must distinguish the agent's actual edit from the structural router check. A passing `check` cannot prove semantic placement or complete repository verification.

Focused tests cover invalid and overlapping roots (`internal/format/yaml_documentation_test.go`); participating directories, direct-child coverage, extra cross-links, missing and unsafe targets, reference links, code examples and normalization (`internal/app/documentation_routers_test.go`); fixed-snapshot `check` and broken-router `verify` (`cmd/markitect/documentation_routers_test.go`); and the executable placement fixture (`examples/example_test.go`). These tests assert structural behavior. The separate agent exercise below probes the authoring decision.

One isolated authoring exercise used a copy of the checked-in [placement example](../examples/documentation-placement/README.md) and the prompt recorded there. The agent changed only `docs/engineering/persistence.md`, the existing page whose README claims schema migration requirements; it created no second page. The router check passed on that candidate. This is one observed placement decision, not a measured reliability rate or proof that the prose is correct.

This slice introduces no Resource Kind, documentation graph, README generation, excludes, or additional Project options. Router navigation and explicit dependency/impact analysis remain separate.

The v0.6.0 source, tag, Windows/Linux release gates, attested assets and post-publication benchmark are recorded in the [production assessment](production-assessment.md).

## Published maintainer tooling (v0.7.0)

The published v0.7.0 release includes `markitect-release distribution` to check or regenerate
marked README installation sections and versioned WinGet manifests from a
verified published immutable release. It validates the exact provenance workflow
attempt, tag, asset bytes, and attestations before generating local files.
[Operations](operations.md) owns the commands and submission checks. This does
not publish a WinGet catalog entry.

## v0.7.0 — fixed task and selected artifact context

The published v0.7.0 release adds a fixed-run context request for one explicitly selected resource, committed work-item snapshot, and bounded exact project artifact paths. The command requires a full Git revision and a manifest from that revision; its report binds included and missing selections, hashes, and completeness to the selected snapshot. It does not infer task relevance or validate acceptance criteria. See [Project artifact inputs](documentation.md) for the run contract. An adopting repository owns the task snapshot, artifact selection, and any expected-resource inventory preflight.

The published source, Windows/Linux gates, versioned assets, provenance and owner publication are recorded in the [production assessment](production-assessment.md). The immutable v0.7.0 release has four verified assets, and its post-publication README used the attested Windows/Linux digests.

## v0.8.0 — repository layout

The published v0.8.0 release recommends `.markitect/areas/<owner>/` for canonical typed knowledge, organized by responsibility. New `init` plans use `.markitect/areas/<namespace>` when `--path` is omitted; explicit paths and existing Projects remain supported. Embedded authoring distinguishes typed Areas from human-owned documentation and uses configured ownership before the new-project convention. Optional kind-suffixed names do not infer YAML type or identity. The executable [layout example](../examples/repository-layout/README.md) checks new paths and generated native outputs while existing fixtures retain earlier layouts.

[Repository layout](repository-layout.md) owns the compatibility assessment. Generic sibling companions, provider output paths, Project schema, root agent instructions and pinned distribution/bootstrap paths retain their contracts. A separate companion-output design, distribution migration, optional layout lint, and migration helper are deferred until demonstrated needs justify their compatibility costs. The reviewed source, Windows/Linux release gates, immutable publication and attested assets are recorded in the [production assessment](production-assessment.md). README installation examples and versioned WinGet metadata use the verified release. Existing consumers must explicitly upgrade their installed tools or project pins.

## v0.8.1 — resolved snapshot boundary

The published v0.8.1 release gives resolved project state a concrete value in `internal/snapshot`: an ID, provisional status, file bytes, and regular-file mode tokens. Parsing, context, review decisions, and impact consume resolved values. Snapshot comparison deterministically reports added, modified, and removed paths from bytes and modes. Git revision resolution, commit-tree reads, and working-tree acquisition stay in `internal/source`; materialization writes the selected value to a filesystem tree without consulting Git. Git-specific write guards remain in the use cases and source adapter that need repository safety. Release bundling and content-package source provenance retain their Git contracts.

The public CLI continues to accept its existing Git revision selectors such as branches and `HEAD`; the fixed-run manifest and stored review-evidence workflows retain their full commit ID requirement. YAML continues to use its existing `revision` field, and package pins produced by the CLI continue to use `git:<full-commit-id>`. Review application logic can compare fixed opaque IDs, while the existing CLI enforces full Git commit evidence before resolving those workflows. Snapshot IDs do not enter the legacy content digest. The existing provisional directory reader remains; this release adds no new fixed directory or archive provider, selector syntax, or generic provider registry. See [Source snapshots](source-snapshots.md) for the complete boundary and compatibility inventory, and the [production assessment](production-assessment.md) for the verified source, tag, platform gates and release assets.

## v0.9.0 — separated outputs and consolidated tool pins

The published v0.9.0 release deliberately replaces the sibling Markdown and consumer distribution contracts. `Project.spec.targets` selects `markdown` explicitly to render deterministic views and local generated navigation under `docs/markitect/<area>/`; a Project without that target requires no generic Markdown output. Codex and Claude projections route directly to canonical YAML or explicitly declared original files and operate independently of generic views. Canonical Area paths remain explicit and configurable; filename suffixes do not infer resource kind or identity.

Consumer bundles place the complete pin set under `.markitect/tool/` (`lock.yaml`, `release.yaml`, `source.zip`) and `.markitect/bootstrap/` (`run.go`, `run_test.go`). The source package command uses the same tool path, and content-package pin suggestions use `.markitect/packages/`. Caches remain outside the declarative tree. Installer, bootstrap and bundle verification retain digest, source provenance, exact-file-set and repository write protections. This release introduces one new bundle contract rather than legacy layout modes or an automatic migration helper.

Generated files retain exact ownership, deterministic links and drift/stale checks. They cannot become ordinary `spec.files` inputs. Existing sibling outputs must be reviewed and removed explicitly when adopting the new contract. Root project instructions and human documentation remain project-owned. Initialization still creates only the Project and one Area README.

Executable fixtures demonstrate both optional Markdown views and provider-only operation. Benchmark fixture v1 remains immutable; v2 covers the new layout. The release-transition benchmark records each binary's compatible fixture and suppresses direct performance-change percentages when fixture versions differ. [Repository layout](repository-layout.md) and [Operations](operations.md) own the implementation and release contracts. The reviewed source, Windows/Linux gates, immutable publication, verified public downloads and versioned transition benchmark are recorded in the [production assessment](production-assessment.md).

## v0.9.1 — source-relative prose navigation

The published patch corrects copied resource prose whose relative links broke when v0.9.0 moved Markdown views. Prose URLs are authored relative to canonical YAML: generic Markdown views map exact local typed YAML destinations to their selected view, while provider inline text keeps canonical destinations. Ordinary document/image links are rebased without changing their target. Query strings, fragments, reference links and code examples are covered by focused tests; link projection uses explicit snapshot files to distinguish ordinary Markdown from an old generated companion. It infers no dependencies or input declarations. [Repository layout](repository-layout.md#source-relative-prose-navigation) owns the contract. The reviewed source, Windows/Linux gates, immutable release, verified public downloads and same-fixture native benchmark are recorded in the [production assessment](production-assessment.md).

## Later product options

Current source adds a neutral [onboarding exercise](onboarding.md), its replay in Windows CI, and structural fixture checks on both supported platforms. [WinGet distribution](winget.md) records the submitted v0.5.0 portable manifests and local installation/upgrade checks; submission is distinct from publication in Microsoft's catalog. The [MCP evaluation](mcp-evaluation.md) records the concluded bounded assessment, the decision to keep MCP experimental, and the deferred second-client criterion. The MCP prototype remains experimental and is excluded from published CLI binaries.

- MCP, LSP, and graph visualization when measured authoring needs justify them.
- A long-running background operator beyond the explicit v0.10.0 observe/plan/apply/verify commands only if users show a need those bounded operations do not meet.

These options are deferred product decisions, not commitments of the current release. Keep the core provider-independent and add abstractions only for a demonstrated invariant.

<a id="markitect-first-release-candidate"></a>
## Markitect-first (published v0.13.0)

The [canonical Change workflow](../internal/host/embedded/resources/workflow-markitect-first-change.yaml), [usage](usage.md#markitect-first-release-candidate), root Project and generated provider Skills make the engineering-entrypoint direction executable. The artifact helper uses existing checks and explicit path accounting rather than new Core/Domain semantics. The [release-readiness assessment](validation/markitect-first-release-readiness.md) records scope and gate evidence. v0.13.0 is the published additive minor release; v0.12.0 remains immutable historical evidence.

The exact owner-approved private Konfyra capture has been written outside both repositories and four bounded interpretation candidates prepared. Public documentation contains only sanitized counts/tool behavior. Candidate acceptance, canonical adoption and actual recurring human burden remain owner decisions; no adoption or productivity result is implied. Prior pilot negative findings remain intact. After generic release gates, the next product validation is a published-binary, owner-reviewed real-task comparison including initial setup and recurring maintenance, using [Measurement](measurement.md).

The owner-supplied [research refinement](design/projection-assurance-direction.md) sets expressiveness/projection/verification/human-attention questions without introducing Obligations, AI-verifier hierarchies or broader Core primitives. Highest-value follow-up remains an owner-reviewed real slice, including check/model duplication and maintenance cost.
