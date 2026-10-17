# First companion slice: architecture, data and deployment

Date: 17 October 2026. Concrete proposal requested by the owner after the [concept-closure plan](standard-library.md#path-to-concept-closure). The owner selected the earlier system/application/deployment distinctions and commissioned this exact scope discussion; the six Kind contracts, Package assignments and example below are recommendations pending selection. This document owns their proposed wording and the connected example. The [discussion record](standard-library.md) owns selections and provenance; the [data-and-view contract](standard-library-data-and-views.md) owns detailed transaction, migration, recovery and C4 recommendations.

This prospective Web Library does not replace the local Library M1/M2 fixture. These are authoring examples, not compiled inputs, deployed infrastructure or execution evidence.

## Recommended scope and distribution

Add exactly six Kinds. Keep the selected eight foundation Kinds unchanged.

| Distribution model | Package | First Kinds |
| --- | --- | --- |
| markitect-architecture | markitect/architecture | System, Application |
| markitect-data | markitect/data | DataStore |
| markitect-platform | markitect/deployment | Environment, Deployment, OperatingResource |

Retain the previously proposed coherent model boundaries rather than put every topic into the foundation. Each companion owns one populated Package initially; do not create empty http, messaging, networking, operations or c4 Packages in anticipation of later content. A Package can grow compatibly under its model's release policy; new specialized Packages need an actual distinct contract and owner. A small initial Package is acceptable; the cost is three independently pinned companion releases.

Each proposed model starts at preview 0.1.0 when actually published, follows the selected immutable/exact-pin release policy and has a direct dependency on markitect-core 1.0.0. The six Kind files below import only Core. Thus the companion models need no artificial dependency on one another or on markitect-standard. Combining their Kinds in a project does not create library-model dependencies; actual cross-model references would require them. Core is acquired separately, never copied into these models. No versions or lock entries are published by this proposal.

| Consumer | Direct model dependencies in this example |
| --- | --- |
| Each companion | markitect-core: 1.0.0 |
| Web Library | markitect-standard: 0.1.0; markitect-architecture: 0.1.0; markitect-data: 0.1.0; markitect-platform: 0.1.0; markitect-core: 1.0.0 if retaining its own LendingPolicy Kind/Properties. |

The actual consumer binds every transitive dependency through its complete lock and explicit acquisition. There is no implicit model download. Model IDs, exact versions and Package imports remain different concepts. A consumer imports only the Packages used by that Definition.

All six proposed Kinds use body links and `spec: {}`. No structured consumer currently justifies environment, replicas, engine, provides/consumes or resource-reference fields. This is a proposal for these six contracts, not an empty-spec policy for all future companions. The body obligations below are assessed semantically; empty specs do not make their sentences compiler-enforced relationship types. V1 supplies neither inheritance nor a union-reference mechanism.

## Complete proposed Kind definitions

Each code block is one Definition file. The paths are relative to that distribution's model root and establish Package membership; there is no extra package declaration.

### System

File: markitect/architecture/system.yaml.

```yaml
imports:
  - package: markitect/core
    as: Core
id: System
kind: Core.Kind
body: >-
  Describes a coherent software solution, its purpose, scope,
  responsibilities and external boundary.
  Identifies its included parts and relevant external dependencies
  through explicit links and states their relationship to the solution.
  Its boundary does not follow automatically from repository,
  model, Package or Manager boundaries.
spec: {}
```

### Application

File: markitect/architecture/application.yaml.

```yaml
imports:
  - package: markitect/core
    as: Core
id: Application
kind: Core.Kind
body: >-
  Describes an executable software unit with an intended runtime boundary,
  purpose and responsibilities, independently of a particular deployment.
  Identifies relevant offered or consumed interaction contracts and
  deliberately distinguished internal parts through explicit links.
  Frontend and backend may describe its role.
  It does not identify a particular running process or replica.
spec: {}
```

### DataStore

File: markitect/data/data-store.yaml.

```yaml
imports:
  - package: markitect/core
    as: Core
id: DataStore
kind: Core.Kind
body: >-
  Describes a logical data-holding boundary and its responsibility
  for the data identified through explicit links.
  States the applicable access, consistency and data-lifetime expectations
  directly or through linked contracts.
  Domain meaning remains with the referenced domain definitions.
  The logical boundary does not identify a database engine,
  hosting service, physical volume or particular runtime instance.
spec: {}
```

### Environment

File: markitect/deployment/environment.yaml.

```yaml
imports:
  - package: markitect/core
    as: Core
id: Environment
kind: Core.Kind
body: >-
  Describes an intended operating context and its purpose,
  applicable conditions and constraints.
  States relevant isolation and integration boundaries directly
  or through linked contracts.
  It does not duplicate application or domain meaning
  and does not report the current observed runtime state.
spec: {}
```

### Deployment

File: markitect/deployment/deployment.yaml.

```yaml
imports:
  - package: markitect/core
    as: Core
id: Deployment
kind: Core.Kind
body: >-
  Describes the intended arrangement of one Application or logical
  DataStore in one Environment, identifying both through explicit links.
  Identifies the OperatingResources used and states their relationship
  to the subject, with relevant capacity, configuration,
  placement and integration constraints.
  Distinguishes where software executes from how it is delivered.
  It describes desired arrangement, not an executed rollout
  or an inventory of current processes.
spec: {}
```

This explicitly proposes extending the previously accepted application-only subject to logical DataStores. One subject and one Environment make a Deployment independently addressable. A deployment may use several resources and describe several roles/replicas; split it when an independently meaningful arrangement needs its own identity. The prose alternative Application or DataStore is not a typed union field. A named OperatingResource is already an intended capability in its stated scope; it need not receive another Deployment just to repeat that declaration.

### OperatingResource

File: markitect/deployment/operating-resource.yaml.

```yaml
imports:
  - package: markitect/core
    as: Core
id: OperatingResource
kind: Core.Kind
body: >-
  Describes an intended technical resource or service capability
  used to operate or deliver software or data.
  States its capability, intended operating or sharing scope,
  permitted use and relevant operational responsibility boundaries.
  Identifies relevant consumers, resources or contracts through
  explicit links and states applicable lifetime and failure expectations.
  It does not duplicate application behavior or logical data meaning
  and does not represent a transient observed runtime object.
spec: {}
```

An API runtime, web-delivery service, database service, router/load balancer or separately governed persistent storage can satisfy this contract. Do not introduce DatabaseService, LoadBalancer, Volume and Node Kinds merely as classifications. Their capabilities and constraints can be stated here. A future specialized Kind needs additional reusable obligations or a demonstrated structured consumer.

An Application owns project software behavior and an executable boundary. An OperatingResource owns an operating capability and responsibility boundary. Both may involve executable software in reality: an external database engine is modeled here for its hosting/service capability; its project-specific data meaning stays with DataStore. A custom proxy whose behavior is itself developed may instead be an Application with its own Deployment on runtime resources. Do not create two Definitions of the same subject just to give it two labels.

## Connected Web Library example

Use project Package library for the solution and software/data contracts, and library/production for this operating arrangement. Existing domain Concepts and UseCases remain in their feature Packages. This folder split also creates Manager responsibilities under the accepted Manager design; it is a deliberate example choice, not a requirement of any new Kind. System containment is still stated explicitly.

The following are complete envelopes of a connected architecture slice, one block per file. Imported members/member, catalog/copy and lending/loan, borrow and return denote the corresponding domain Definitions retained from the Library. Their complete behavior, the other Library operations, security design, HTTP protocol, checks and data/recovery Rules remain part of the complete reference project and are not replaced by these short architecture bodies. These declarations alone are not a complete production specification.

### Solution and logical software/data subjects

File: library/library.yaml.

```yaml
imports:
  - package: markitect/architecture
    as: Architecture
id: library
kind: Architecture.System
body: >-
  Supports the Library's operator workflows for members, copies and loans.
  Includes [[library-web]], [[library-api]] and [[library-data]].
  Operator browsers execute the frontend; the project delivers its software.
  Browser devices and underlying provider infrastructure are outside the
  project-operated boundary unless explicitly constrained.
spec: {}
```

File: library/library-web.yaml.

```yaml
imports:
  - package: markitect/architecture
    as: Architecture
id: library-web
kind: Architecture.Application
body: >-
  The frontend executes in an operator's browser and presents the Library
  workflows through [[library-http-api]]. It does not access
  [[library-data]] directly. Delivery of its assets is distinct
  from execution in the browser.
spec: {}
```

File: library/library-api.yaml.

```yaml
imports:
  - package: markitect/architecture
    as: Architecture
id: library-api
kind: Architecture.Application
body: >-
  The backend provides [[library-http-api]] and contains [[lending-logic]].
  All its runtime occurrences use [[library-data]] through
  [[library-data-access]]; per-process state must not become an
  independent authority for members, copies or loans.
spec: {}
```

File: library/lending-logic.yaml.

```yaml
imports:
  - package: markitect/software
    as: Software
  - package: lending
    as: Lending
id: lending-logic
kind: Software.Component
body: >-
  Realizes [[Lending.borrow]] and [[Lending.return]] within the API runtime.
  Uses [[library-data-access]] for coordinated decisions and state changes.
  It does not have a separate process or deployment boundary.
spec: {}
```

File: library/library-data.yaml.

```yaml
imports:
  - package: markitect/data
    as: Data
  - package: members
    as: Members
  - package: catalog
    as: Catalog
  - package: lending
    as: Lending
id: library-data
kind: Data.DataStore
body: >-
  The authoritative logical store for [[Members.member]], [[Catalog.copy]]
  and [[Lending.loan]] state. Their meaning remains with those Concepts.
  Access and coordinated state changes follow [[library-data-access]].
  Accepted changes outlive replacement of API instances.
  Concrete engine placement and recovery arrangements belong
  to the deployment and operating contracts.
spec: {}
```

File: library/library-http-api.yaml.

```yaml
imports:
  - package: markitect/software
    as: Software
id: library-http-api
kind: Software.Interface
body: >-
  The HTTP interaction contract consumed by [[library-web]].
  Defines operation mapping, request/response and refusal behavior,
  correlation, access requirements and failure outcomes for the
  supported operator workflows. Domain behavior remains with its UseCases.
spec: {}
```

File: library/library-data-access.yaml.

```yaml
imports:
  - package: markitect/software
    as: Software
id: library-data-access
kind: Software.Interface
body: >-
  Governs access to [[library-data]] by [[lending-logic]] and other
  API responsibilities. Admission and resulting writes form one
  coordinated operation across API instances. Success requires
  confirmed durable commit; known aborts leave no partial changes;
  an uncertain commit is reported without blind automatic replay.
  Subsequent inspections observe completed changes in a coherent state.
spec: {}
```

The HTTP body identifies the scope of a contract still needing its actual protocol and security clauses; it is not a substitute for those clauses. The access body summarizes the owning [data contract](standard-library-data-and-views.md#2-data-access-and-the-admission-boundary). In an authored complete model, move detailed obligations into their single owning Interface or linked Rules, rather than maintaining both this example summary and another binding copy. A data adapter can realize the access contract without introducing another public API or mandatory Component split.

### Environment, resources and desired arrangements

All files below belong to Package library/production. Every cross-Package reference imports library explicitly; parent-folder membership supplies no implicit import.

File: library/production/production.yaml.

```yaml
imports:
  - package: markitect/deployment
    as: Platform
id: production
kind: Platform.Environment
body: >-
  The operating context for the production Library service.
  Production data and credentials are isolated from local development
  and recovery exercises. Access, configuration and recovery obligations
  apply to the resources and arrangements that explicitly target it.
spec: {}
```

File: library/production/api-runtime.yaml.

```yaml
imports:
  - package: markitect/deployment
    as: Platform
id: api-runtime
kind: Platform.OperatingResource
body: >-
  Provides project-controlled server execution capacity in [[production]].
  Supports the intended occurrences required by [[api-production]].
  The project controls configuration, health assessment and replacement
  requirements. Host counts and placement remain realization choices
  until an explicit failure-domain requirement constrains them.
spec: {}
```

File: library/production/database-service.yaml.

```yaml
imports:
  - package: markitect/deployment
    as: Platform
id: database-service
kind: Platform.OperatingResource
body: >-
  Provides database execution and retained storage in [[production]]
  for [[data-production]]. The project retains responsibility for
  access, configuration and recovery requirements, whether hosting
  is operated directly or delegated through a provider contract.
  API replacement must not discard retained database state.
  Engine, durability, migration and recovery capabilities must satisfy
  the logical store and its applicable contracts.
spec: {}
```

File: library/production/web-delivery.yaml.

```yaml
imports:
  - package: markitect/deployment
    as: Platform
id: web-delivery
kind: Platform.OperatingResource
body: >-
  Provides delivery of browser application assets in [[production]]
  for [[web-production]]. The project controls the intended published
  assets and their compatibility requirements. Asset delivery does
  not execute the frontend and grants no direct database access.
spec: {}
```

File: library/production/public-entry.yaml.

```yaml
imports:
  - package: markitect/deployment
    as: Platform
  - package: library
    as: Library
id: public-entry
kind: Platform.OperatingResource
body: >-
  Provides the project-controlled HTTP routing and load-balancing
  boundary in [[production]] for [[Library.library-http-api]].
  Routes only to suitable healthy occurrences of [[api-production]].
  Failure handling must preserve the interface's outcome contract;
  uncertain state-changing requests must not be blindly replayed.
  This resource does not provide shared lending state.
spec: {}
```

File: library/production/api-production.yaml.

```yaml
imports:
  - package: markitect/deployment
    as: Platform
  - package: library
    as: Library
id: api-production
kind: Platform.Deployment
body: >-
  Arranges [[Library.library-api]] in [[production]] on [[api-runtime]].
  Requires at least two interchangeable API runtime occurrences.
  Their logical data access is bound to the store arrangement
  [[data-production]]. Runtime configuration uses references to the
  required connection settings and credentials, not live secret values.
spec: {}
```

File: library/production/data-production.yaml.

```yaml
imports:
  - package: markitect/deployment
    as: Platform
  - package: library
    as: Library
id: data-production
kind: Platform.Deployment
body: >-
  Arranges [[Library.library-data]] in [[production]] on [[database-service]].
  All occurrences of [[api-production]] address the same authoritative
  logical state. Technology and schema/configuration bindings must
  satisfy [[Library.library-data-access]] and the applicable migration
  and recovery contracts. Capacity does not imply automatic failover.
spec: {}
```

File: library/production/web-production.yaml.

```yaml
imports:
  - package: markitect/deployment
    as: Platform
  - package: library
    as: Library
id: web-production
kind: Platform.Deployment
body: >-
  Makes [[Library.library-web]] available in [[production]]
  through [[web-delivery]]. The application executes in operator
  browsers; its API endpoint is [[public-entry]]. The delivered
  frontend and [[api-production]] must share a compatible
  [[Library.library-http-api]] contract.
spec: {}
```

The two API occurrences are a desired capacity constraint, not two separately maintained Instance Definitions and not proof of high availability. Resources do not silently imply failure-domain separation, zero data loss or a particular host count. A managed database can encapsulate persistence; declare a separate volume only when its lifecycle, placement or retention has an independently relevant contract. API and data arrangements may link to each other without creating an execution order or dependency in the Manager process.

PostgreSQL remains the earlier reference recommendation, not a choice embedded in generic DataStore or OperatingResource meaning. If selected, its precise engine/configuration requirement belongs to database-service and its deployment bindings, with migration/recovery obligations at their own linked owners. Complete the reference model's protocol/security clauses, concrete configuration requirements and acceptance Checks before treating it as a web implementation assignment. Implementation then supplies source/lock bindings and realizes the selected contracts. This does not block selecting the generic six-Kind vocabulary. No empty body clause should be read as permission to invent a production security or recovery policy.

## Relationship ownership and examination

| Statement | Single owning Definition in this slice |
| --- | --- |
| Web/API/store belong to the solution | library |
| Frontend consumes HTTP and cannot access the store directly | library-web |
| API offers HTTP and contains lending logic | library-api |
| Lending logic realizes borrow/return | lending-logic |
| Store owns authoritative domain state | library-data |
| Interaction and transaction/outcome semantics | library-http-api and library-data-access, respectively |
| Production isolation | production |
| Runtime, data-service and asset-delivery capabilities | api-runtime, database-service and web-delivery, respectively |
| Healthy API routing and load-balancer replay constraints | public-entry |
| API capacity, execution placement and environment binding | api-production |
| Store-to-database-service placement and environment binding | data-production |
| Frontend delivery, browser execution distinction and endpoint binding | web-production |

Linking the same subject from several owners is normal when they state different facts. Do not repeat the same mutable fact in reciprocal lists. These are binding sentences plus target identities, not formal contains/hosts/routes operators. A semantic Check must examine their meaning; the compiler verifies reference/field validity only.

This slice supplies the subjects for the earlier C4 mapping without new C4 identities. Do not infer that web-delivery executes the browser Application, that an OperatingResource always means a C4 deployment node, or that an Application automatically satisfies every view's required boundary. The [view contract](standard-library-data-and-views.md#1-c4-views-of-one-model) owns generation, provenance and independent assessment.

Review should be able to answer where the frontend executes, what API replicas share, which object owns domain meaning, which resource hosts the store, what would change for another environment, and which relationship supports each diagram edge. Local link/accounting checks do not supply this semantic assessment or runtime evidence.

## Decision proposed to the owner

Select these six Kinds in the three listed Packages, initially with body links and empty specs. Select Deployment's single Application-or-DataStore subject in one Environment and keep OperatingResource as the common first operating-capability contract. Defer dedicated Instance, DatabaseService, LoadBalancer and Volume Kinds until their extra contract is demonstrated. Keep C4 as reviewed views of these same objects. Exact project security/protocol, PostgreSQL, production operating targets and physical distribution are separate choices, not implicitly selected by accepting these generic definitions.

Return to the [standard-library discussion](standard-library.md) or [design map](README.md).
