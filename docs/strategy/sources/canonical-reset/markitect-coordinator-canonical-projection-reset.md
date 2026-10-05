# Markitect — Canonical Model / Projection Architecture Reset

## Coordinator brief

Continue Markitect from the **current `main` branch**.

Before changing anything, inspect the repository as it exists now. Do not assume older strategy documents, previous prompts, the published release, or earlier module boundaries are still the desired architecture.

At minimum inspect:

- `README.md`
- `docs/architecture.md`
- `docs/engineering-constitution.md`
- `docs/implementation-plan.md`
- `docs/strategy/`
- `docs/usage.md`
- the current schema / type-system implementation
- the normalized model and graph implementation
- current plan/apply/reconciliation logic
- current module contracts and module boundaries
- adoption/import logic
- artifact coverage logic
- agent-rule / skill generation
- Markdown/documentation generation
- Git/GitHub/Azure DevOps/pipeline/hook integrations
- .NET integration
- tests and examples
- any work currently present from the previous A/B gauntlet or projection-first experiments

Treat current code and docs as **implementation evidence**, not automatically as canonical product requirements.

This document supersedes earlier design directions where they conflict with it.

Do not preserve an old primitive, abstraction, file format, module boundary, or workflow merely for compatibility if it contradicts the architecture below.

The objective is not to add more features.

The objective is to make Markitect internally coherent around one much stronger product model:

> **Markitect is the canonical desired-state model of an engineering system. Everything else is a projection or an observation of that canonical intent.**

---

# 1. Product thesis

Software implementation is becoming cheap because AI agents can generate large amounts of code and configuration.

Human attention is not becoming cheap.

The risk is therefore no longer primarily that implementation is expensive. The risk is that cheap implementation creates large amounts of locally plausible but globally inconsistent software, documentation, automation, agent guidance, and operational configuration.

Markitect exists to make engineering intent explicit, canonical, typed, addressable, projectable, and verifiable.

The central direction is:

> **Define intent once. Project it everywhere. Verify every representation. Reconcile drift.**

A second formulation:

> **Humans program the engineering system. Agents program its projections.**

And the operational rule:

> **AI should implement the architecture, not reinvent it on every task.**

The core product metric is:

> **More autonomous agent work per unit of human attention while preserving deliberately designed engineering boundaries.**

---

# 2. Canonical authority

Markitect is the **single semantic source of truth**.

This is stronger than “Markitect helps synchronize documentation.”

The intended model is:

```text
                       MARKITECT
                   canonical intent
                         │
        ┌────────────────┼────────────────┐
        ▼                ▼                ▼
      source           docs             CI/CD
      tests            skills           hooks
      config           pipelines        agent files
        │                │                │
        └────────────────┼────────────────┘
                         ▼
                    projections
```

Code, Markdown, CI files, Skills, AGENTS files, Git hooks, pipeline YAML, infrastructure configuration, generated schemas, and similar artifacts are **representations of canonical Markitect intent**.

They are not independent competing authorities.

If a projection disagrees with Markitect, the normal operation is:

```text
canonical intent unchanged
        ↓
projection is wrong
        ↓
reconcile the projection
```

Do **not** silently change the canonical model to match drifted code or generated artifacts.

If the actual desired design changes, then:

```text
human/authorized agent changes canonical intent
        ↓
compile
        ↓
impact analysis
        ↓
reproject affected representations
        ↓
verify
```

This distinction between **intent change** and **projection drift** is fundamental.

---

# 3. Markitect Core

The Core must become deliberately small.

The Core should not contain DDD, workflows, Rules, Projection semantics, Assurance concepts, .NET concepts, AI concepts, or product-specific architecture.

Its responsibility is:

> **Provide a deterministic compiler and type system for canonical engineering Definitions.**

The Core answers:

> **Is this canonical model structurally valid, unambiguous, type-correct, and referentially consistent?**

It does not answer:

> Is this architecture good?

or:

> Does this implementation semantically satisfy the business intent?

Those are higher-level concerns.

---

# 4. Minimal Core language

The current target Core language is:

```text
Definition
Schema
Kind
Property
type
reference
kindReference
minCount
maxCount
```

This is intentionally small.

Do not add additional primitives without proving that the higher-level model cannot be expressed cleanly without them.

---

# 5. Definition

A `Definition` is the universal canonical model unit.

> **A Definition states that something exists in the desired engineering model, what kind of thing it is, what it is called, why it exists, and what is declared about it.**

Universal shape:

```yaml
apiVersion: software.example.org/v1
kind: UseCase

metadata:
  namespace: orders
  name: create-order

purpose: >
  Allows a customer to create an order.

spec:
  ...
```

Universal fields:

```text
apiVersion
kind
metadata.namespace
metadata.name
purpose
spec
```

Every Definition must have a `purpose`.

The intent is:

> **If something deserves canonical existence, its reason for existence should be explicit.**

`purpose` should describe responsibility / reason for existence, not incidental implementation trivia.

---

# 6. Definition identity

Definition identity is:

```text
apiVersion
+
kind
+
namespace
+
name
```

Conceptually:

```text
DefinitionIdentity(
  apiVersion,
  kind,
  namespace,
  name
)
```

There is no need for an additional opaque instance UUID initially.

Changing identity is a remove + add at Core level.

Higher-level migration semantics may later describe continuity.

Git remains responsible for source history.

Do not build a parallel generic version-control system into Markitect.

---

# 7. Schema

A `Schema` is a versioned collection of Kind declarations.

Example:

```text
Schema:
software.example.org/v1

Kinds:
  Module
  UseCase
  Handler
  Validator
```

`Schema` replaces earlier use of `Domain` as a generic Core container.

`Domain` is too semantically loaded because DDD may legitimately define its own Domain concept.

`apiVersion` identifies the semantic Schema/version under which a Definition is interpreted.

---

# 8. Kind

A `Kind` describes one nominal category of Definition.

Example:

```text
Kind: UseCase

Purpose:
Represents one independently understandable unit
of application behavior.
```

A Kind declares its Properties.

Kind identity is conceptually:

```text
apiVersion + kind
```

The Core does not understand what `UseCase`, `Module`, `Rule`, `Aggregate`, or any other project vocabulary means.

It only understands the Kind contract.

---

# 9. Property

A Property defines one allowed piece of information inside a Definition's `spec`.

A Property should minimally declare:

```text
name
purpose
type
minCount
maxCount
```

Example:

```text
Property: transactional

Purpose:
States whether successful execution must be atomic.

type:
boolean

minCount:
1

maxCount:
1
```

Properties are part of a Kind declaration.

They are not required to be globally addressable Definitions.

---

# 10. Property types

Prefer conventional names.

The initial candidate set is:

```text
string
boolean
integer
number
enum
object
reference
kindReference
```

Do not introduce a generic type system.

Do not introduce:

```text
Ref<T>
List<T>
Optional<T>
Map<K,V>
```

Multiplicity is orthogonal:

```text
exactly one:
  minCount: 1
  maxCount: 1

optional:
  minCount: 0
  maxCount: 1

many:
  minCount: 0
  maxCount: unbounded
```

---

# 11. Reference

`reference` is a Property type that points to another concrete Definition.

Example Kind Property:

```yaml
belongsTo:
  purpose: >
    Identifies the Module that owns the UseCase.

  type: reference

  target:
    apiVersion: architecture.example.org/v1
    kind: Module

  minCount: 1
  maxCount: 1
```

A Reference is not a string convention.

The compiler resolves it to a real Definition identity and verifies the nominal target Kind.

---

# 12. References form the semantic graph

Do not add a separate Core `Relation` primitive.

The Property name is the semantic relation.

Example:

```text
CreateOrder.belongsTo = Orders
```

means:

```text
CreateOrder --belongsTo--> Orders
```

Likewise:

```text
CreateOrder --handledBy--> CreateOrderHandler
CreateOrder --validatedBy--> CreateOrderValidator
```

The compiler should derive the typed semantic graph from reference-valued Properties.

---

# 13. kindReference

`kindReference` is the only still-slightly-provisional part of the minimal Core.

It means:

> A reference to a Kind, not to a concrete Definition.

Example use cases now exist:

```text
Rule appliesTo Kind Module
ProjectionPolicy appliesTo Kind Aggregate
Verifier capability supports Kind UseCase
```

This is useful when a canonical statement applies to an entire class of Definitions.

Do not remove it merely because it initially looks unusual.

However, explicitly pressure-test it while implementing higher layers.

If the same requirements can be represented more cleanly without a first-class Kind reference, propose removal before cementing it.

---

# 14. Object values

`object` is for local structured values without independent semantic identity.

Rule of thumb:

> **If something needs its own purpose, identity, relations, reuse, ownership, or independent lifecycle, it should probably be a Definition rather than an embedded object.**

Avoid building large anonymous object trees that recreate an untyped second ontology inside `spec`.

---

# 15. Core exclusions

The following are **not Core primitives**:

```text
Domain
Module
UseCase
Command
Query
Handler
Validator
Aggregate
Rule
Invariant
Workflow
Skill
Agent
Process
Projection
ProjectionPolicy
Obligation
Evidence
Decision
Exception
Authority
C#
Java
Go
GitHub
Azure DevOps
Claude
Codex
```

They may be modeled as Kinds in higher-level Schemas.

Importance does not imply Core status.

---

# 16. No inheritance or generic OO type system

Do not add:

```text
extends
implements
inheritance
subtyping hierarchy
trait hierarchy
multiple inheritance
overrides
default methods
```

to the initial Core.

Composition + explicit Properties + References are preferred.

If real implementation later proves reusable semantic contracts are necessary, solve that requirement explicitly rather than importing OO complexity preemptively.

---

# 17. No Invariant DSL in Core

Do not encode engineering semantics into a Core constraint language merely because some checks are deterministic.

An `Invariant` can be a normal higher-level Kind.

Example conceptually:

```yaml
apiVersion: markitect.foundation/v1
kind: Invariant

metadata:
  namespace: architecture
  name: module-independence

purpose: >
  Preserve independent module evolution.

spec:
  appliesTo:
    apiVersion: architecture.example.org/v1
    kind: Module

  statement: >
    A Module must not depend directly on another
    Module's private implementation.
```

The Core validates its structure.

A later verifier determines whether the statement is true in a projection.

Verification may be:

```text
Core type check
static analyzer
architecture test
behavior test
AI semantic verifier
human decision
```

Intent and verification mechanism are separate.

---

# 18. Core compiler responsibility

The compiler should validate at least:

```text
Definition identity
Kind existence
mandatory purpose
closed spec
known Properties
Property value types
minCount / maxCount
enum values
reference resolution
reference target Kind
kindReference resolution
duplicate Definition identities
duplicate Kind identities
deterministic normalization
```

It should produce at least:

```text
Normalized Model
Resolved Definitions
Typed Reference Graph
Provenance
Diagnostics
```

Core remains deterministic.

AI does not participate in deciding whether the structural model compiles.

---

# 19. Three conceptual layers

Keep this separation explicit.

## Level 1 — Core / structural truth

Question:

> Is the canonical model valid and unambiguous?

Mechanism:

```text
Definition
Schema
Kind
Property
type
references
compiler
```

## Level 2 — Engineering intent

Question:

> What should exist and what should be true?

Possible Kinds:

```text
Project
Goal
Rule
Invariant
Process
ProjectionPolicy
UseCase
Module
Aggregate
Dossier
...
```

These are built with Core.

## Level 3 — Projection and assurance

Question:

> How is intent materialized, and what evidence says the materialized reality still satisfies it?

Mechanisms:

```text
Projectors
deterministic tools
AI agents
tests
analyzers
verification agents
run records
```

---

# 20. Agent-led UX

The primary user experience should be a **Markitect-aware AI agent**.

The user should not need to know the schema syntax before using Markitect.

Typical entry:

> Initialize this repository as a Markitect project.

The agent should actively guide the user.

It should ask about:

```text
project goal
product/system type
architecture
technology
deployment/platform
documentation needs
delivery process
agent tooling
```

It should know the available Module registry and recommend suitable Modules.

The normal interaction is:

```text
human discussion
      ↓
Markitect-aware agent
      ↓
proposed canonical model
      ↓
Core compiler
      ↓
accepted canonical state
```

Direct editing remains possible, but is not the primary UX assumption.

---

# 21. Markitect is not Markdown-first

Do not rebuild a workflow in which Markdown is a second canonical source.

Markdown is a projection target.

For example:

```text
Markitect
   ↓
Markdown Projector
   ↓
docs/architecture.md
docs/domain.md
docs/workflows.md
```

If documentation drifts, regenerate/reconcile it from Markitect.

A canonical Definition may still contain natural-language Properties such as:

```text
purpose
rationale
statement
guidance
```

Natural language is valid canonical data.

The difference is that the canonical text lives in the Markitect model, not duplicated manually across many target files.

---

# 22. What is a Markitect Module?

A Module is:

> **A versioned installable extension package that provides reusable Schemas, Projectors, or both.**

A Module may also provide discovery metadata and reusable recommendations/profiles.

Conceptually:

```text
Module
├── identity/version
├── purpose
├── provided Schemas        optional
├── provided Projectors     optional
├── discovery metadata      optional
└── reusable profiles       optional
```

At least one useful capability must exist.

Do not force every Module to provide both language and projection.

---

# 23. Language Modules

Examples:

```text
DDD
Vertical Slice
CQRS
Clean Architecture
```

A DDD Module might provide:

```text
BoundedContext
Aggregate
Entity
ValueObject
DomainEvent
Repository
```

A Vertical Slice Module might provide:

```text
UseCase
Command
Query
Handler
Validator
```

Such Modules may have no Projector at all.

Their value is reusable vocabulary.

---

# 24. Projection Modules

Examples:

```text
.NET
Markdown Documentation
Azure DevOps
Git Hooks
GitHub
Codex
Claude
```

A Markdown Module may mostly provide Projectors.

A .NET Module may provide Projectors and perhaps some technology-specific Schemas.

Azure DevOps may provide both canonical Kinds such as Pipeline/Stage/Environment and Projectors.

Do not decide this based on symmetry.

Let each Module provide what is actually useful.

---

# 25. Module isolation

This is a hard architectural principle:

> **Modules synchronize with Markitect, never with each other.**

Desired architecture:

```text
                 Markitect Model
                /       |        \
               /        |         \
            .NET     Markdown    Azure DevOps
```

Not:

```text
.NET ↔ Markdown ↔ Azure DevOps ↔ Codex ↔ Git
```

A Module works against:

```text
1. canonical Markitect model
2. its own target representation
```

It must not treat another Module's output as semantic authority.

Do not create pairwise synchronization logic.

---

# 26. Vocabulary and Projector decoupling

Avoid this:

```text
.NET knows every DDD concept
DDD knows every programming language
```

That produces an ontology × target cross-product.

A Projector may consume canonical context supplied by Markitect, but should not require unrelated Module implementations or synchronize with them.

Prefer reusable ontology + project-specific projection intent + target Projector.

---

# 27. ProjectionPolicy / projection intent

A key higher-level concept is likely needed to bridge abstract canonical intent and a specific target technology without coupling Modules.

Working name:

```text
ProjectionPolicy
```

This belongs above Core, probably in a Foundation Schema.

Example:

```yaml
apiVersion: markitect.foundation/v1
kind: ProjectionPolicy

metadata:
  namespace: architecture
  name: aggregate-to-dotnet

purpose: >
  Defines how DDD Aggregates should normally be represented
  in this project's .NET implementation.

spec:
  sourceKind:
    apiVersion: ddd.example.org/v1
    kind: Aggregate

  target: dotnet

  guidance: >
    Represent Aggregates as behavior-rich domain classes.
    Keep persistence concerns outside the domain type.
    Do not use EF attributes in domain classes.
```

This gives:

```text
DDD semantics
    +
project-specific projection policy
    +
.NET target knowledge
    ↓
valid .NET representation
```

The DDD Module does not need to know .NET.

The .NET Module does not need hard-coded DDD behavior.

For AI Projectors, the Projector can interpret semantic Definitions + ProjectionPolicy + target context.

For deterministic Projectors, stronger explicit contracts may be required.

The exact final name/schema may change, but preserve the architectural separation.

---

# 28. Projector

A Projector is:

> **A Module-provided capability that materializes a bounded canonical Markitect scope into one target representation.**

Examples:

```text
dotnet-source
markdown-documentation
azure-pipeline
git-hooks
codex-bootstrap
```

A Projector may be:

```text
deterministic code
templates
AI agent
combination of deterministic and AI mechanisms
```

---

# 29. Projector contract

A Projector should expose enough information for Markitect to plan and execute it without special-case knowledge.

At minimum pressure-test a contract containing:

```text
identity
purpose
target kind/type
allowed target surface
applicability
required canonical context
supported selectors / semantic scope
projection guidance / execution entrypoint
target inspection needs
verification guidance
```

Do not overdesign the schema before implementing several real Projectors.

The contract must be sufficient for:

```text
projection discovery
context construction
work planning
artifact ownership
verification
impact analysis
```

---

# 30. Projector target surface

A Projector should declare where it is allowed to operate.

Example conceptually:

```yaml
target:
  type: repository

  surface:
    - "src/**"
    - "tests/**"
    - "*.sln"
    - "*.csproj"
```

This is an **allowed surface**, not automatic ownership of every file matching the pattern.

Actual artifact ownership is recorded by projection runs.

---

# 31. Projector input

A Projector run should receive bounded context.

Conceptually:

```text
canonical revision
canonical scope
resolved dependencies
applicable Rules / Invariants
applicable ProjectionPolicies
currently owned artifacts
relevant target state
```

It should not automatically receive:

```text
entire repository
all Definitions
all Modules
all docs
all policies
```

unless the scope genuinely requires them.

---

# 32. Projector output

A Projector must report what it changed and what it now claims as part of its projection.

Conceptually:

```text
ProjectionResult

created artifacts
modified artifacts
deleted artifacts
retained owned artifacts
diagnostics
failure
escalation
```

A Projector must not make unreported side effects part of normal operation.

The exact write model — direct changes vs patch/apply — may be chosen during implementation.

The architectural requirement is traceability.

---

# 33. Projectors must not invent canonical intent

A Projector has freedom over implementation details inside the allowed solution space.

It does **not** have authority to silently add new engineering intent.

Example:

A .NET projection agent discovers that it would like retry semantics.

It must not simply add a retry mechanism if that changes intended behavior.

It should return something like:

```text
ESCALATION

Projection requires a new engineering decision:
Should CreateOrder have retry semantics?
```

Then return to canonical modeling.

After canonical intent changes:

```text
compile
→ impact
→ replan
→ project again
```

---

# 34. Non-deterministic projection is intentional

Traditional generation assumes:

```text
R = f(M)
```

Markitect should allow:

```text
R ∈ ValidRepresentations(M)
```

Two agents may produce different valid implementations of the same intent.

This is not a problem.

The key question is whether each representation satisfies declared canonical semantics and required verification.

This is one of the central reasons AI makes the Markitect approach practical.

---

# 35. Projection Record

Projection execution must create durable operational provenance.

Working concept:

```text
ProjectionRecord
```

This is **not canonical engineering intent**.

It is derived/operational state describing what was actually materialized.

Conceptual record:

```yaml
projectionId: ...

projector:
  module: markitect-dotnet
  id: dotnet-source
  version: 1.4.2

canonicalRevision: abc123

scope:
  - software/v1:UseCase:orders/create-order

policies:
  - markitect.foundation/v1:ProjectionPolicy:architecture/aggregate-to-dotnet

artifacts:
  created:
    - path: src/Orders/CreateOrderCommand.cs
      digest: ...

    - path: src/Orders/CreateOrderHandler.cs
      digest: ...

  modified:
    - path: src/Orders/OrdersModule.cs
      digest: ...

previousProjection: ...

result:
  status: succeeded
```

Do not copy this exact syntax blindly.

Implement the semantics.

---

# 36. Projection Ledger

Projection Records together provide a bidirectional ledger:

```text
Canonical Definition / Scope
        ↓
Projection
        ↓
Artifacts
```

and:

```text
Artifact
   ↓
Projection
   ↓
Canonical Definition / Scope
```

Markitect should eventually be able to answer:

> Why does this file exist?

Example:

```text
src/Orders/CreateOrderHandler.cs

owned by:
dotnet-source projection

canonical scope:
CreateOrder

canonical revision:
abc123
```

This traceability is essential for drift control.

---

# 37. Artifact ownership

Every relevant technical artifact should be classified.

Current candidate states:

```text
MANAGED
IGNORED
EXCLUDED
UNKNOWN
```

## MANAGED

Owned by a known Markitect projection.

## IGNORED

Technically irrelevant generated/environmental material, for example:

```text
.git/
bin/
obj/
node_modules/
```

## EXCLUDED

Explicitly outside Markitect management, for example an intentionally unmigrated legacy area.

Exclusion must remain visible.

Do not treat it as “does not exist.”

## UNKNOWN

Relevant artifact with no known projection owner and no explicit exclusion.

This should produce diagnostics.

Example:

```text
artifact.unowned

src/Orders/MagicOrderFix.cs
has no known projection owner.
```

The system should help determine whether it should:

```text
belong to an existing projection
cause new canonical modeling
be explicitly excluded
or be deleted
```

---

# 38. Artifact coverage

Artifact ownership gives a concrete coverage dimension.

Example:

```text
relevant artifacts: 137

managed:   131
excluded:    6
unknown:     0
```

`unknown: 0` should be a desirable state.

Do not reduce the entire Markitect assurance model to one coverage percentage, but artifact coverage is an important operational measure.

Excluded artifacts should remain inspectable and visible in assurance output.

---

# 39. Projection Record vs Verification Result

Do not combine materialization provenance and verification evidence into one giant record.

Keep the conceptual responsibilities separate:

```text
Projection Record
= what was materialized from what canonical scope

Verification Result
= what was checked, how, and with what result
```

A single projection may be checked by:

```text
compiler
static analyzer
unit tests
integration tests
AI verifier
higher-level composition verifier
```

without rerunning the projection.

---

# 40. No first-class Observer for now

Do not introduce an `Observer` abstraction unless real implementation proves it necessary.

Projectors and Verifiers naturally need to inspect current target state.

For now:

```text
target inspection
= operation used by projection / verification
```

not:

```text
Observer
= mandatory architectural role
```

Keep the vocabulary small.

---

# 41. Greenfield initialization

The expected new-project flow is:

```text
markitect init
      ↓
empty Markitect project
      ↓
small Foundation available
      ↓
agent asks for project goal/context
      ↓
capture initial canonical intent
      ↓
query Module Registry
      ↓
recommend Modules
      ↓
user selects Modules
      ↓
Schemas + Projectors become available
      ↓
guided modeling discussion
      ↓
canonical model grows
      ↓
project when sufficiently specified
```

Do not require the user to manually know which Modules to install before describing the project.

The agent should guide discovery and selection.

---

# 42. Foundation should remain small

A default Foundation Module is reasonable so that a brand-new Markitect project can record basic intent before architecture Modules are selected.

Likely concepts may include things such as:

```text
Project
Goal
Rule
Invariant
ProjectionPolicy
```

but do not blindly implement this exact list.

Derive the smallest useful Foundation from the workflows being implemented.

Foundation is not Core.

---

# 43. Existing project adoption

Brownfield adoption is a first-class workflow.

Do not treat it as a secondary import utility.

Desired process:

```text
initialize/adopt
      ↓
inventory repository
      ↓
detect technologies / structures
      ↓
suggest Modules
      ↓
user confirms Modules
      ↓
analyze existing project
      ↓
infer candidate canonical model
      ↓
user + agent review/correct
      ↓
accept selected intent as canonical
      ↓
establish artifact ownership
      ↓
classify managed / ignored / excluded / unknown
      ↓
resolve unknowns
      ↓
verify and reconcile
```

Critically:

```text
existing repository
      ↓
inference
      ↓
candidate model
```

NOT:

```text
existing repository
      ↓
canonical truth
```

Existing code is evidence from which candidate intent can be inferred.

It does not automatically become authority.

---

# 44. Module discovery

Modules may provide cheap discovery metadata so Markitect can recommend them for an existing repository.

Example:

```yaml
discovery:
  signals:
    - type: file
      pattern: "**/*.csproj"
      confidence: high

    - type: file
      pattern: "**/*.sln"
      confidence: high
```

Examples:

```text
.NET:
*.sln
*.csproj
global.json
Directory.Build.props

Go:
go.mod
go.sum

Azure DevOps:
azure-pipelines*.yml
.azuredevops/
```

Semantic applicability such as DDD may require a deeper AI-assisted analysis.

Preferred flow:

```text
cheap deterministic discovery
        ↓
optional semantic AI discovery
        ↓
user confirmation
```

Do not create a large Detector framework unless the implementation demonstrates a need.

---

# 45. Module Registry

Markitect needs a registry/catalog capability sufficient to:

```text
discover Modules
read purpose
read provided Schemas
read provided Projectors
read compatibility/version metadata
read discovery metadata
recommend Modules
install/enable selected Modules
resolve required dependencies
```

The agent should use this registry during both greenfield initialization and brownfield adoption.

The exact distribution mechanism can be decided during implementation.

Do not overbuild a remote marketplace before proving the local/package model.

---

# 46. Recursive execution and verification

Do not use one giant agent with the instruction:

> Implement everything and then check it.

Markitect should decompose work into small scopes.

Every scope uses the same two conceptual roles:

```text
Executor
Verifier
```

There is no separate `Integrator` role.

The process repeats recursively at larger scopes.

---

# 47. Executor

The Executor is responsible for bringing one bounded scope toward canonical desired state.

At a leaf this may mean:

```text
create one Handler
project one document
update one pipeline fragment
```

At a higher scope it may mean:

```text
repair how already-valid children compose
connect several projections
adjust local integration
```

The role is the same.

Only scope changes.

The Executor may be:

```text
deterministic tool
AI agent
or a combination
```

---

# 48. Verifier

The Verifier independently checks one bounded scope.

Desired loop:

```text
Executor
   ↓
Verifier
   ├── PASS
   └── FAIL + bounded feedback
            ↓
         Executor
            ↓
         Verifier
            ↓
           ...
```

The loop ends with:

```text
PASS
```

or explicit escalation/failure.

The Verifier should receive:

```text
canonical intent for the scope
relevant dependencies/contracts
current projected state
verification criteria
relevant child results/evidence
```

It should not simply reuse the Executor's reasoning and declare its own work correct.

Preserve separation of responsibilities.

---

# 49. Recursive verification upward

Example:

```text
CreateOrder
├── Handler
├── Validator
└── Endpoint
```

First:

```text
Handler:
Executor → Verifier → PASS

Validator:
Executor → Verifier → PASS

Endpoint:
Executor → Verifier → PASS
```

Then run the **same** process at the parent scope:

```text
CreateOrder:
Executor → Verifier
```

The higher-level Verifier checks composition:

```text
Does validation actually protect mutation?
Does the endpoint invoke the intended UseCase?
Do the child contracts fit together?
Does the complete UseCase satisfy its canonical purpose?
```

Then continue upward:

```text
leaf
  ↓
UseCase
  ↓
Module
  ↓
Product
  ↓
System
```

No special Integrator role is needed.

---

# 50. Canonical graph, execution DAG

The canonical Markitect model is a graph, not necessarily a tree.

Shared dependencies are normal.

Example:

```text
CreateOrder ─┐
             ├──> Order
CancelOrder ─┘
```

Do not distort ontology into a tree merely to simplify execution.

Instead:

```text
Canonical Model
= typed graph
```

and:

```text
Execution Plan
= derived DAG / hierarchical work plan
```

The execution view may appear tree-like for a selected root scope while preserving shared dependencies correctly.

---

# 51. Scoped context

Every Executor and Verifier should receive only the context needed for its scope.

A leaf Executor may receive:

```text
target Definition
direct dependencies
applicable Rules / Invariants
ProjectionPolicy
Projector instructions
currently owned artifacts
relevant local target files
```

A higher-level Verifier may receive:

```text
parent Definition
child contracts
child verification results
composition requirements
relevant canonical policies
```

Avoid loading:

```text
entire repository
entire canonical model
all documentation
all Rules
all Modules
```

by default.

Small scoped context is a core economic and reliability advantage.

---

# 52. Impact analysis

When canonical intent changes, use the graph + projection ledger to determine what must rerun.

Conceptually:

```text
changed Definitions
      ↓
semantic dependencies
      ↓
affected ProjectionPolicies
      ↓
affected Projection Records / Projectors
      ↓
affected execution scopes
      ↓
required parent verification scopes
```

Only rerun the affected subgraph and necessary upward verification path.

Do not fall back to global “read the whole project and check everything” unless explicitly requested or technically unavoidable.

---

# 53. Reconciliation

When verification fails but canonical intent is unchanged:

```text
Verifier FAIL
      ↓
bounded diagnostics
      ↓
Executor repairs projection
      ↓
Verifier reruns
```

Do not change canonical intent to make verification pass.

If the failure reveals missing/incorrect intent, escalate back to modeling.

---

# 54. Escalation

A projection/execution task must stop and escalate when it cannot proceed without inventing new canonical intent.

Examples:

```text
new business behavior required
new architecture responsibility needed
new cross-module dependency decision
new security policy needed
ambiguous ownership
incompatible canonical Rules
```

The escalation returns to the Markitect-aware modeling workflow.

Once the canonical model is updated and compiled, planning resumes.

---

# 55. Convergence

A selected scope is converged when:

```text
required projections have completed
required verification has passed
required recursive parent-scope checks have passed
unknown required artifacts are resolved
no unresolved escalation remains
```

Do not define convergence as:

> The software is objectively correct or bug-free.

Define it as:

> **The declared engineering assurance contract for this scope is currently satisfied against this canonical revision.**

---

# 56. Capabilities required above Core

Treat these as **capabilities first**, not automatically as separate services, Kinds, or Modules.

Required capabilities currently include:

```text
Project initialization
Foundation bootstrap
Module registry/catalog
Module discovery
Module selection/resolution
Guided modeling
Core compilation
Model querying
Model diff
Impact analysis
Projection discovery
ProjectionPolicy resolution
Work planning
Context building
Execution orchestration
Verification orchestration
Recursive scope verification
Projection record persistence
Artifact ownership indexing
Artifact inventory/coverage
Reconciliation
Escalation
Run/evidence recording
Convergence evaluation
Brownfield inference/adoption
```

Only promote a capability into a new ontology Kind or architectural primitive when the implementation proves that it needs stable addressable semantics.

---

# 57. Important distinction: canonical vs operational state

Markitect as a system knows more than the canonical engineering model.

Keep these separate.

## Canonical state

What should be true.

Examples:

```text
Definitions
Rules
Invariant statements
ProjectionPolicies
architecture responsibilities
project goals
```

## Operational / derived state

What happened while trying to realize it.

Examples:

```text
ProjectionRecords
artifact digests
run IDs
verification results
timestamps
diagnostics
current artifact ownership index
```

Do not put operational bookkeeping into canonical Definitions simply because Markitect needs to store it.

---

# 58. Suggested Module manifest direction

Do not freeze syntax before trying real modules, but a useful candidate contract is:

```yaml
name: markitect-dotnet
version: 1.0.0

purpose: >
  Provides projection capabilities for .NET repositories.

requires:
  core: ">=1.0"

provides:
  schemas: []
  projectors:
    - dotnet-source

discovery:
  signals:
    - type: file
      pattern: "**/*.sln"
      confidence: high
    - type: file
      pattern: "**/*.csproj"
      confidence: high
```

DDD may look like:

```yaml
name: markitect-ddd
version: 1.0.0

purpose: >
  Provides reusable Domain-Driven Design vocabulary.

provides:
  schemas:
    - ddd/v1

  projectors: []
```

Azure DevOps may provide both.

Use these as design examples, not frozen file formats.

---

# 59. Existing Markitect modules

Current module names/boundaries are not sacred.

The repository previously contained concepts/modules around areas such as:

```text
Adoption
Agent Rules
Markdown
Artifact Coverage
Git Hooks
Pipelines
.NET
GitHub
Azure DevOps
```

Re-evaluate each against the new architecture.

Some may remain useful as Modules.

Some may become Projectors.

Some may become generic host capabilities.

Some may disappear because the canonical/projection architecture subsumes them.

Do not retain old module boundaries merely because tests already exist.

Preserve useful behavior and evidence, not accidental architecture.

---

# 60. Old A/B gauntlet work

Preserve previous benchmark evidence and invalid-cohort notes as historical evidence.

Do not let the old benchmark harness dictate the new architecture.

The earlier DDD experiment revealed an important failure mode:

```text
implementation changed
canonical model did not
guidance-first / sidecar architecture allowed divergence
```

That failure is one of the motivations for this reset.

Do not resume holdout comparisons using the old assumptions until the new canonical/projection architecture is coherent enough to test fairly.

---

# 61. Implementation approach

Do not attempt a giant rewrite in one uncontrolled commit.

Use a staged migration.

The exact phases may change after inspecting current `main`, but a reasonable sequence is:

## Phase 0 — Reconcile design with current repository

Produce a concise architecture delta:

```text
keep
change
remove
replace
defer
```

Map current concepts to the new design.

Identify compatibility constraints that are genuinely valuable vs accidental.

## Phase 1 — Minimal Core

Prove:

```text
Schema
Kind
Property
Definition
reference graph
minimal compiler
```

with no DDD or product-specific concepts in Core.

Remove or isolate Core concepts that violate the new boundary.

## Phase 2 — Small Foundation + Module contract

Provide just enough Foundation to support:

```text
project intent
rules/invariants if needed
ProjectionPolicy if needed
```

Define a real Module manifest/registration model and at least two very different example Modules.

## Phase 3 — Projector contract

Implement a generic Projector registration/execution contract.

Use at least:

```text
Markdown Projector
.NET Projector or another code Projector
```

to pressure-test it.

Do not special-case them in Core.

## Phase 4 — Projection Ledger / Artifact Ownership

Implement:

```text
ProjectionRecord
artifact ownership
managed / ignored / excluded / unknown
coverage diagnostics
```

Prove bidirectional traceability.

## Phase 5 — Greenfield flow

Prove:

```text
init
capture project goal
recommend modules
select modules
model intent
project a small scope
verify
```

The first version may use local/static registry metadata.

Do not block on a production remote registry.

## Phase 6 — Brownfield adoption

Prove:

```text
repo inventory
module discovery
candidate model inference
human confirmation boundary
ownership establishment
unknown artifact detection
```

## Phase 7 — Recursive Executor/Verifier orchestration

Implement the smallest credible orchestration model that can:

```text
derive scoped work
execute
verify independently
repair
repeat upward
```

Use fake/deterministic agents where necessary to test orchestration before binding everything to a specific AI provider.

## Phase 8 — Impact/reconciliation

Prove that a small canonical change reruns only affected projections and required parent verification scopes.

---

# 62. Required architectural tests / invariants

Add tests that make the new architecture hard to accidentally violate.

At minimum protect:

```text
Core does not depend on Foundation/application Modules
Core has no DDD/.NET/Markdown/Azure-specific semantics
Module projectors do not call/synchronize other projectors
canonical model and operational projection state are separate
reference graph derives from typed reference Properties
unknown Definition Properties fail compilation
invalid references fail compilation
duplicate Definition identities fail
ProjectionRecords bind to canonical revision
artifact ownership is traceable both directions
unknown relevant artifacts surface diagnostics
projection drift does not mutate canonical intent
```

If the implementation architecture supports compile-time dependency tests, use them.

---

# 63. Stress scenarios

Do not declare the design successful merely because the classes compile.

Use real scenarios.

## Scenario A — Greenfield software project

Start empty.

Describe:

```text
modular .NET application
DDD
Vertical Slice
Azure DevOps
Markdown documentation
```

Markitect should:

```text
capture initial project intent
suggest relevant Modules
enable selected Modules
allow modeling
project at least one UseCase
project documentation
record artifact ownership
verify
```

## Scenario B — DDD Aggregate → .NET

Model an `Aggregate`.

Do not hard-code DDD logic into the .NET Module.

Use project-specific ProjectionPolicy/guidance.

Show that the .NET Projector can produce one valid representation.

## Scenario C — Markdown documentation

Project the same canonical intent to Markdown.

Markdown must not inspect .NET output to learn semantic truth.

It reads Markitect.

## Scenario D — Projection variation

Run two projection attempts that produce different implementation details.

Both should be acceptable if they satisfy the same canonical intent and verification.

This proves:

```text
R ∈ ValidRepresentations(M)
```

rather than fixed template generation.

## Scenario E — Hidden artifact

Add an unowned behavior-relevant file manually.

Markitect should detect it as `UNKNOWN`.

It must not disappear behind a generic ignore rule.

## Scenario F — Explicit exclusion

Mark a legacy area as intentionally excluded.

It remains visible in coverage/assurance output.

## Scenario G — Intent change

Change canonical `CreateOrder` intent.

Impact analysis should find affected projections and required parent verification scopes.

Unrelated areas should not rerun.

## Scenario H — Projection drift

Modify a managed generated/projected file without changing canonical intent.

Markitect should identify drift and reconcile the projection.

It must not update the canonical model to match the file.

## Scenario I — New semantic decision required

During projection, force a situation where the implementation agent wants a new architectural/business decision.

The Projector/Executor must escalate instead of inventing intent.

## Scenario J — Brownfield repository

Initialize Markitect in an existing repository.

Detect likely technology Modules.

Infer candidate intent.

Require review before canonicalization.

Establish artifact ownership and expose remaining unknowns.

---

# 64. User experience goals

The user should be able to work approximately like this:

```text
User:
Initialize this as a Markitect project.

Agent:
What are you building?

User:
A modular .NET application for ...

Agent:
Based on your goals and repository, I recommend:
- DDD
- Vertical Slice
- .NET
- Azure DevOps
- Markdown Documentation

Would you like to use these?

User:
Yes.

Agent:
Let's model the product responsibilities first...
```

Then normal architecture discussion continues.

The user should not spend most of their time hand-authoring Markitect YAML.

The agent should understand Markitect, guide the user, propose model changes, compile them, explain diagnostics, and use installed Modules.

---

# 65. Universal agent protocol

Provider-specific bootstrap should be thin.

A Markitect-aware agent should effectively follow:

```text
1. Treat Markitect as canonical semantic authority.

2. Guide the user toward suitable Modules and explicit intent.

3. Record accepted engineering intent in Markitect.

4. Do not create a competing canonical source in code/docs/agent files.

5. Before changing engineering intent, update/propose Markitect first.

6. Before projection, obtain bounded canonical context.

7. Work only inside the current Projector/scope.

8. Do not synchronize directly with other Modules.

9. If target state conflicts with Markitect, Markitect wins unless intent is explicitly changed.

10. Do not silently invent new architecture/business semantics during projection.

11. Escalate missing intent.

12. Return projection provenance and verification results.

13. Use verification feedback to repair the projection, not to rewrite canonical intent.
```

---

# 66. Naming discipline

Prefer obvious established words.

Current preferred vocabulary:

```text
Definition
Schema
Kind
Property
type
reference
kindReference
Module
Projector
ProjectionPolicy
ProjectionRecord
Executor
Verifier
```

Avoid new terms when an established term is clearer.

Do not introduce abstractions such as:

```text
Observer
Integrator
Relation
Shape
ValueType
```

unless implementation produces a concrete requirement that cannot be expressed clearly with the current vocabulary.

---

# 67. What to leave open

The Coordinator has latitude on implementation details.

Do not overconstrain:

```text
exact YAML syntax
exact file layout
database vs files for operational records
event log vs current-state index
exact CLI command names
exact registry transport
exact Module version solver
exact AI provider integration
exact patch/apply mechanism
exact retry policy
exact confidence formula for discovery
exact internal WorkNode class hierarchy
```

Choose the simplest implementation that preserves the architecture.

Document significant deviations and why they are necessary.

---

# 68. What is NOT open

Do not reinterpret these away:

```text
Markitect is the canonical semantic source of truth.

Core stays minimal and generic.

Documentation/code/CI/agent files are projections, not competing truth.

Modules do not synchronize with each other.

Modules may provide Schemas, Projectors, or both.

Projection-specific cross-technology decisions belong in canonical projection intent/policies,
not hard-coded ontology × target coupling.

Projectors do not invent canonical intent.

Projection provenance/artifact ownership must be explicit.

Unknown relevant artifacts must be visible.

Canonical model is a graph.

Execution/verification is scoped and recursive.

Executor and independent Verifier are the recurring roles.

No separate Integrator role.

Intent change and projection drift are different workflows.

Brownfield adoption infers candidate intent; existing code is not automatically canonical.

AI is used above the deterministic Core, not inside Core compilation.
```

---

# 69. Deliverables

Work autonomously from current `main`.

Do not stop after producing design prose.

Deliver:

```text
1. repository/design assessment against this brief

2. updated canonical architecture documentation

3. updated implementation plan

4. coherent minimal Core migration

5. Module contract/registration implementation

6. at least representative Schema-only and Projector-oriented Modules

7. Projector contract

8. ProjectionRecord / artifact ownership mechanism

9. unknown/excluded/managed artifact accounting

10. greenfield initialization path

11. initial brownfield/adoption path

12. scoped execution/verification architecture

13. tests for architectural boundaries and core semantics

14. realistic examples proving the design

15. migration notes from the previous architecture

16. a clear list of remaining hypotheses / deliberately deferred questions
```

If full implementation of every later orchestration feature is too large for one coherent change, implement the foundations and the smallest end-to-end vertical slice that proves the architecture, then continue in staged PRs.

Do not stop at interfaces with no executable proof.

---

# 70. Decision method during implementation

Whenever a new abstraction appears necessary, ask:

```text
Is this:
1. universal structural semantics?
2. reusable engineering intent?
3. Module/package capability?
4. Projector-specific behavior?
5. operational derived state?
6. merely an implementation detail?
```

Use the answer to place it correctly.

The bar for adding something to Core is especially high:

> **Multiple unrelated ontologies must require exactly this universal semantics, and the requirement cannot be expressed cleanly using the current Core.**

Do not add Core primitives because a single Module would find them convenient.

---

# 71. Falsify the design while implementing

Actively try to disprove this architecture.

In particular look for cases where:

```text
Definition + Schema + Kind + Property types are not expressive enough

kindReference creates more complexity than value

ProjectionPolicy becomes effectively code-generation DSL

Module isolation forces duplication or prevents necessary composition

artifact ownership cannot represent shared/multi-projector files cleanly

recursive scoped verification costs more than global review

AI Projectors cannot reliably respect semantic boundaries

Foundation grows into a second hard-coded Core

Brownfield inference requires too much manual reconstruction

operational ProjectionRecords become a second authority

impact analysis is too broad to save work
```

If one of these happens, preserve the evidence and propose the smallest correction.

Do not hide negative findings.

---

# 72. Research-quality evidence

This redesign is also a research hypothesis.

Instrument examples/tests where practical.

Useful measurements include:

```text
canonical model size
number of Definitions
number of Projectors
artifact ownership coverage
unknown artifact count
projection scope size
context size per agent
number of retries
verification failures caught
impact radius after change
how many unrelated scopes avoided rerunning
human interventions / escalations
```

Do not optimize prematurely, but preserve enough run data to evaluate whether the architecture actually reduces human attention and global context requirements.

---

# 73. Final implementation principle

The system should converge toward this:

```text
                    HUMAN / ARCHITECT
                           │
                           ▼
                  Markitect-aware Agent
                           │
                           ▼
                  CANONICAL DEFINITIONS
                           │
                           ▼
                    CORE COMPILER
                           │
                           ▼
                 TYPED SEMANTIC GRAPH
                           │
            ┌──────────────┼──────────────┐
            ▼              ▼              ▼
         Projector      Projector      Projector
           .NET         Markdown        Azure
            │              │              │
            ▼              ▼              ▼
         Executor       Executor       Executor
            │              │              │
            ▼              ▼              ▼
         Verifier       Verifier       Verifier
            │              │              │
            └──────────────┼──────────────┘
                           ▼
             larger-scope Executor/Verifier
                           │
                      repeat upward
                           │
                           ▼
                       CONVERGED
```

With:

```text
Markitect
= canonical desired engineering state

Modules
= reusable language and/or projection capabilities

Projectors
= materializers of bounded canonical intent

ProjectionRecords
= provenance of what was materialized

Artifact Ownership
= what projection owns what representation

Executor
= performs projection/reconciliation work

Verifier
= independently evaluates a bounded scope

Impact Analysis
= determines what must rerun after canonical change

Reconciliation
= brings divergent projections back to canonical intent
```

The intended end state is not a better collection of AI instruction files.

It is a system in which:

> **Engineering intent is modeled once, AI receives only the context it needs, every representation has provenance, each scope is independently verified, and changes propagate through known semantic dependencies instead of global synchronization runs.**

Start by inspecting current `main`, produce the architecture delta, then implement the smallest coherent path toward this model.
