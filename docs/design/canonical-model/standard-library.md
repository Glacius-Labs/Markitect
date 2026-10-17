# Standard-library design

Status: consolidated design for integration, 17 October 2026. The owner selected fourteen Kinds, their Package assignments, body links, empty specs, the version policy and the six project-coverage areas. This document owns the current scope, distribution/version contract and implementation handoff. The [language specification v1](minimal-language/language-contract.md) remains the sole language contract. No library release or implementation is supplied by these documents.

## Selected scope and owning definitions

| Distribution model | Package | Kinds and owning definition text |
| --- | --- | --- |
| markitect-standard | markitect/software | [Concept](concept.md), [Rule](rule.md), [UseCase](use-case.md), [Actor](actor.md), [Interface](interface.md), [Component](component.md) |
| markitect-standard | markitect/verification | [Scenario](scenario.md), [Check](check.md) |
| markitect-architecture | markitect/architecture | [System and Application](standard-library-architecture-platform.md#complete-selected-kind-definitions) |
| markitect-data | markitect/data | [DataStore](standard-library-architecture-platform.md#datastore) |
| markitect-platform | markitect/deployment | [Environment, Deployment and OperatingResource](standard-library-architecture-platform.md#environment) |

The first eight Kinds form the foundation; the six companions supply the selected architecture/data/deployment extension. Each Kind is an ordinary Core.Kind Definition with one source file. The linked owners contain the consolidated wording. The library adds reusable meaning, not compiler primitives, inheritance, execution authority or local Managers for imported foreign Packages.

## Body links and empty specs

All fourteen Kinds declare `spec: {}`: they admit no custom structured fields. Definitions whose exact kind is one of them also have empty specs; the mapping is not an open extension object. Identity, kind, body, links, imports and optional title remain available under the common envelope.

A body link identifies its target; the surrounding sentence states the relationship, scope and obligation. It does not imply calls, containment, hosting, routing, execution order or authority merely because two subjects are linked. The compiler checks language structure and reference validity; semantic review and relevant Checks assess the intended meaning.

Do not add reciprocal or parallel lists such as UseCase.rules, Component.parts or Interface.providers. In the Library, borrowing directly links membership eligibility, copy availability and the project-specific lending policy. LendingPolicy is not a Rule subtype; a Rule-only list would omit an applicable contract. A custom Kind may still declare useful structured Properties. V1 supplies no nominal inheritance or union-reference target.

### Internal contract and Component boundary

Interface covers external and internal technical interaction contracts. A Component may provide or consume one without becoming an Actor. Component owns a system part's responsibility; Interface owns the interaction agreement. A separate Interface is useful for a deliberate independently addressable contract, not automatically for every helper call. Application identifies an executable runtime boundary; a Component need not have one.

An Actor is a role, not an account or permission grant. A Scenario states an expected concrete case. A Check describes examination, observations and success criteria, not an execution result. The existing repository-specific [command-bearing Check](../../../.markitect/model/vocabulary/check.yaml) remains a separate narrower project Kind; it is not silently migrated to the general-purpose Check.

### Selected boundaries

System owns solution purpose/scope and explicit inclusion, independently of repository, Package or Manager boundaries. Application owns executable software behavior. DataStore owns a logical data-holding responsibility; domain meaning remains with the referenced Concepts. OperatingResource owns an operating or delivery capability and responsibility boundary, such as runtime, database service, web delivery or routing.

Environment owns an operating context. Deployment describes the desired arrangement of one Application or logical DataStore in one Environment using the relevant resources. It distinguishes software execution from asset delivery. Frontend/backend are roles; interchangeable replicas can be a capacity statement. A dedicated Instance, DatabaseService, LoadBalancer or Volume Kind remains deferred until it adds a demonstrated contract. Actual process IDs, observations and rollout history remain records.

## Package and version contract

[Packages](minimal-language/language-contract.md#packages) are namespaces; [models](minimal-language/language-contract.md#models) own versioning/distribution. A populated Package has one owning model per bound input set. Do not append content to another model's populated Package, copy Core or create empty speculative Packages.

The two foundation Packages share markitect-standard's release. The three companion models release independently. Their current Kind files import only Core, so each model's only direct library dependency is markitect-core 1.0.0; no companion/foundation dependency is invented solely because a consumer combines their Kinds. Actual cross-model references would require the corresponding direct dependency.

Folder paths under the model root create Package identities. Model ID and physical distribution folder do not create a prefix. Keep one Definition per YAML file, with no kinds/properties/instances subtrees. `markitect` is an unpopulated prefix, not an umbrella import. Imports are explicit per Definition and do not install or reexport other Packages.

### Publication and binding

Use an unpublished draft during development. Publish each first usable preview as `0.1.0` only after its actual files, manifest, dependencies/complete lock and relevant complete consumer pass validation. No release is published here. Preview revisions remain immutable and exactly pinned; incompatible pre-1.0 revisions advance the middle component, for example 0.2.0, with their effects stated.

Target `1.0.0` after the final contract passes relevant consumer validation. Vocabulary stability is independent of shipping full Apply. The library/model version, Markitect executable version, language formatVersion and Core version remain independent. Release numbers are not part of Package/Definition identity.

After 1.0, use this release convention:

| Change | Increment |
| --- | --- |
| Editorial correction preserving obligations, interpretation and accepted inputs | Patch |
| Additive content preserving existing contracts and name resolution | Minor |
| Changed existing meaning/obligations; renamed, moved or deleted identity; any field added to an existing empty-spec Kind | Major |

Body meaning is public contract. A stronger body obligation can invalidate a realization without a structural error. Even an optional field changes an existing empty field contract. Intentional published content revisions receive new immutable versions/digests; source formatting is not grounds to replace a publication.

Adding a Kind is not automatically compatible: an imported Package exposes short IDs even with an alias, so an addition can make a previously unique consumer reference ambiguous. Alias qualification is not alias-only import. Prefer a distinct explicit Package when it prevents inappropriate name-set expansion; otherwise assess the compatibility effect. The compiler neither classifies compatibility nor selects upgrades.

Consumers declare exact direct model dependencies, acquire all transitive sources explicitly and bind the full external closure through the [lock contract](minimal-language/language-contract.md#lock). Direct Core use, such as the Library's own LendingPolicy Kind/Properties, requires a direct Core dependency. A consumer using only library Kinds need not directly depend on Core solely because libraries use it. Bind one version per model ID at a time; aliases do not enable parallel versions. No ranges, latest selector, automatic downloads or automatic upgrades are introduced.

An update changes the exact requirement and full lock, then checks structure and reassesses applicable semantic/realization/evidence effects. Unchanged local files or a version label do not prove compatibility. Physical distribution paths, release tags and packaging automation remain bounded implementation choices.

## Broader project and repository coverage

The owner selected coverage of security, configuration, build, tests, release and operation using the existing fourteen Kinds, leaving concrete project values open. The [coverage walkthrough](standard-library-project-coverage.md) owns the six areas, proposed reference Definitions and distinguishing examinations. Rules express binding conditions/procedures/required outputs; Checks express assessment; Scenarios express concrete expectations. Actual source files and pipelines belong to realizing artifacts and the registry.

This closes the vocabulary question for those six areas in this bounded example, not every possible software project or a production readiness claim. The project must still supply applicable identity/trust, protocol/configuration, delivery compatibility and service/recovery requirements, or state bounded realization freedom. Do not invent universal providers, thresholds, retention periods, role hierarchies or approval gates. More specialized HTTP, messaging, security or engineering libraries may follow a demonstrated repeated contract; they are not part of this first implementation scope.

## Logical data, operating resources and C4 walkthrough

The [connected Web Library](standard-library-architecture-platform.md#connected-web-library-example) owns its illustrative subjects and explicit relationships: browser/API Applications, lending Component, logical DataStore, production Environment, resources and Deployments. Browser software executes in browsers; its delivery resource serves assets. Database service and logical domain store are different responsibilities. Two API replicas require coordinated authoritative data, not unrelated per-process lending state.

The separate [data-and-view contract](standard-library-data-and-views.md) owns detailed Web Library recommendations for atomic admission, known/unknown commit outcomes, retry, migration and recovery. PostgreSQL and the maintenance/restore approach remain reference proposals; numeric operating/recovery objectives remain project requirements. Selection of generic Kinds or six coverage areas does not select all of those project choices.

### C4 mapping without duplicate project objects

First support uses reviewed views of the same Definitions, without parallel C4 copies or a new mandatory C4 Kind. The initial Web Library mapping is:

| Subject | Representation in the appropriate C4 view |
| --- | --- |
| library System | Software system |
| browser/API Applications and logical DataStore | Containers in the logical view |
| lending logic within the API runtime | Component in that application's Component view, where its boundary meets that contract |
| database hosting service | Deployment node in the illustrative production view |
| intended application/store occurrences | Container instances, without mandatory separate canonical Instance Definitions |
| load balancer | Supporting infrastructure in that deployment view |
| Deployments | Supply arrangement facts; they are not automatically deployment-node boxes |

The external C4 [Container](https://c4model.com/abstractions/container), [Component](https://c4model.com/abstractions/component) and [deployment-view](https://c4model.com/diagrams/deployment) contracts constrain a claimed C4 view, not the broad Component Kind or compiler operators. A generating agent/tool and independent review must assess actual source-supported edge meanings, scope and abstraction against the [view contract](standard-library-data-and-views.md#1-c4-views-of-one-model). Reference validity or graph reachability alone does not establish C4 correctness. A fully deterministic semantic renderer needs a separately justified sufficient relationship contract.

## Observe the value of spec

The owner asked to observe whether spec has a useful application and reconsider it if none emerges. Keep two questions distinct: useful structured content versus the cost of writing an empty mapping everywhere. Removing the literal empty wrapper and removing structured contracts altogether are different language changes. Neither is selected now; the completed v1 baseline and fourteen empty-spec contracts remain in effect.

Examine real consumers and observed benefit, not a field's mere existence. LendingPolicy parameters and the repository's command-bearing Check are examples to assess. Configuration validation or deterministic relationship/rendering consumers could justify a future Property proposal; do not invent fields to defend the mechanism. No standing monitor, new study or scheduled work is commissioned.

<a id="path-to-concept-closure"></a>

## Implementation handoff and integration

The current concept has selected scope, Kind wording owners, Package/version policy, relationship boundaries and project coverage. Independent review may refine wording without reopening selected decisions absent new evidence. Concrete Web Library values, physical packaging and executable consumer evidence do not become unresolved language decisions.

The implementation sequence after the Coordinator checkpoint is:

1. Produce the eight foundation Definition files, manifest and complete exact dependency/lock binding; validate them through the actual compiler and a complete consumer.
2. Add the six selected companion contracts under their three owning models/Packages and validate the connected consumer. Do not dispatch the entire earlier topic map.
3. Keep current local Library M1/M2 acceptance intact. Any deliberate vocabulary migration must preserve its complete intent, including operations, wire protocol, build/test requirements and parameterized policy. Do not add the prospective web behavior to that fixture by reclassification.
4. Record observed authoring/realization benefit and concrete spec consumers during use. Compiler success and Definition counts alone do not establish benefit or production readiness.

This PR is documentation integration preparation: use current Main, resolve real overlap, obtain independent complete-diff review, freeze the head and require exact-head hosted CI under the [delivery rules](../../implementation-plan.md#integration). Main integration, implementation, a published library and real Apply evidence remain distinct. Preparing this concept does not dispatch implementation or infer that a Coordinator checkpoint has happened.

## Source and selected decisions

The [discussion history](standard-library-history.md) preserves the owner's original statements, rejected fields and intermediate proposals. The current consolidation follows these selections:

- Foundation: "Okay, dann nehmen wir die acht Bausteine als ersten Umfang", following explicit empty-spec/body-link choices for all eight; Interface also covers internal technical contracts.
- Distribution: "Dann übernehmen wir die Paketstruktur und Versionierung wie vorgeschlagen."
- Companions: "Okay, dann übernehmen wir die sechs Kinds mit Körperlinks und spec: {} sowie die Paketzuordnung."
- Coverage: "Okay, dann halten wir diese Abdeckung so fest und lassen die konkreten Projektwerte offen."
- Consolidation: "Lass uns jetzt das Konzept konsolidieren, die letzten Widersprüche bereinigen und die Integration in main vorbereiten."

The qualitative coverage and generic contracts are selected. Concrete example arrangements, unselected technology and operating values retain their stated role. These documents establish no compiler/consumer execution, release or independent-review result by themselves.

Return to the [canonical design map](README.md).
