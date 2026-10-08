# Markitect — Designing the Core Modeling Language

## Status

Architecture-language design document for review by the implementer.

The goal is to define a **small, powerful, orthogonal modeling language** that Markitect Core can understand and reason about.

The key question is not:

> Which domain-specific Kinds should Markitect support?

The more important question is:

> **What is the smallest semantic kernel from which many engineering domains can be modeled cleanly?**

The intended inspiration comes from:

- programming languages,
- type systems,
- compilers,
- Kubernetes resource modeling,
- declarative desired-state systems,
- DDD,
- Ports & Adapters,
- composition-oriented design.

This document is exploratory. It should be challenged and refined before implementation.

---

# 1. Core design goal

Markitect should avoid becoming a large collection of hard-coded domain concepts.

Instead of putting every future concept into Core:

```text
Module
Product
UseCase
Aggregate
Interface
Release
Threat
Service
Deployment
Agent
Skill
Workflow
...
```

the Core should know a small semantic language that Domain Extensions can use to define those concepts.

Conceptually:

```text
Markitect Core Language
        ↓
Domain Languages
        ↓
Adapters / Projections / Reconciliation
```

The quality of this Core language will strongly determine whether Markitect remains elegant as it grows.

---

# 2. Learn from programming languages

Good programming languages usually expose a small number of orthogonal concepts that compose well.

Relevant ideas for Markitect:

```text
Types / Kinds
Identity
References
Modules / Namespaces
Interfaces / Protocols
Composition
Constraints / Invariants
Static checking
Packages
Intermediate Representation
```

The important lesson is:

> **Do not add a new language feature for every use case. Build a small set of primitives that combine into richer meanings.**

---

# 3. Source syntax should not be the semantic model

Markitect should distinguish between:

```text
Authoring Syntax
```

and:

```text
Semantic Model
```

For example:

```text
YAML
  ↓
Parse
  ↓
Type / Schema Check
  ↓
Reference Resolution
  ↓
Semantic IR
  ↓
Graph / Impact / Plugins / Reconciliation
```

This means YAML is merely one source language.

Long-term other authoring surfaces could theoretically include:

```text
Visual editor
AI-assisted authoring
TOML
JSON
IDE UI
```

without requiring plugins to change.

A plugin should consume the **validated semantic model**, not raw YAML.

---

# 4. Compiler-inspired architecture

A useful long-term pipeline could be:

```text
1. Parse
       ↓
2. Schema / Type Check
       ↓
3. Resolve References
       ↓
4. Build Semantic Graph
       ↓
5. Evaluate Constraints
       ↓
6. Normalize to Semantic IR
       ↓
7. Impact Analysis
       ↓
8. Select affected adapters
       ↓
9. Plan projections / reconciliation
       ↓
10. Verify
```

Each stage should have a clear responsibility.

This should reduce accidental coupling between syntax, model semantics and adapters.

---

# 5. Kubernetes as inspiration

Kubernetes demonstrates how a relatively small resource model can support many domains.

Its general shape:

```yaml
apiVersion:
kind:
metadata:
spec:
status:
```

is useful because it separates:

```text
Resource identity
Desired state
Observed state
Extension model
Reconciliation
```

Markitect should borrow the useful semantics, not copy Kubernetes wholesale.

---

# 6. `apiVersion`

Versioned domain schemas are valuable.

For example:

```yaml
apiVersion: software.markitect.io/v1alpha1
kind: Module
```

This could allow different Domain Extensions to evolve independently.

Examples:

```text
software.markitect.io/v1alpha1
security.markitect.io/v1alpha1
delivery.markitect.io/v1alpha1
ai.markitect.io/v1alpha1
```

The Core should not need to understand the internal meaning of every versioned domain resource.

---

# 7. `kind`

`kind` should remain a nominal type.

Example:

```yaml
kind: Module
```

The type should never be inferred from:

```text
file name
directory name
provider output
```

The domain extension defines what structure and relations are valid for the Kind.

---

# 8. `metadata`

A small universal identity structure remains useful.

Example:

```yaml
metadata:
  name: survey
  namespace: intake
```

Potential universal metadata may include:

```text
name
namespace
labels
annotations only if justified
version/origin metadata where appropriate
```

The Core identity must remain independent of file location.

A conceptual identity:

```text
namespace / kind / name
```

Example:

```text
intake/Module/survey
```

Package/origin identity may qualify that further internally.

---

# 9. `spec`

`spec` represents the canonical desired meaning or configuration of a Resource.

Example:

```yaml
kind: Module

spec:
  axis: intake
  dependencies:
    - core
```

The exact schema is defined by the Domain Extension.

`spec` should remain declarative wherever possible.

---

# 10. `status`

A Kubernetes-like `status` concept may be useful for reconciliation, but should likely not be stored as canonical source.

Conceptually:

```yaml
status:
  conditions:
    - type: DocumentationReconciled
      status: true

    - type: AzureDevOpsReconciled
      status: false
      reason: AreaMissing
```

This gives a clean distinction:

```text
Spec
    what should be true

Status
    what is currently observed
```

Potential architecture:

```text
Canonical source
    contains desired state

Reconciliation evidence / runtime state
    contains observed status
```

Avoid mixing generated observed state into the canonical source files unless there is a very strong reason.

---

# 11. CRD-style extensibility

One of the strongest Kubernetes ideas is that the Core does not understand every resource Kind.

Kubernetes does not need built-in semantic knowledge of:

```text
Certificate
KafkaCluster
PostgresCluster
ServiceMonitor
```

Similarly, Markitect Core should ideally not need built-in knowledge of:

```text
Module
UseCase
Aggregate
Threat
Deployment
Service
```

A domain extension should be able to define new semantic Kinds.

Conceptually:

```text
              MARKITECT KERNEL

Resource
Identity
Reference
Relation
Constraint
Scope
Package
Graph
Change
Reconciliation
        │
        ▼
      Domains
```

---

# 12. Kernel Language vs Domain Language

A useful distinction:

## Kernel Language

Very small universal concepts:

```text
Resource
Kind
Identity
Reference
Relation
Constraint
Scope
Package
```

Potentially:

```text
Contract
Transition
Effect
```

if they prove universal enough.

## Domain Language

Defined using the kernel:

```text
software:
  Module
  Product
  UseCase
  Aggregate
  Interface
  Invariant

ai:
  Agent
  Skill
  Workflow
  Rule

delivery:
  Release
  Environment
  Deployment
```

The Core should ideally not require 40 hard-coded Kinds.

---

# 13. Candidate Core primitive: Resource

Everything with canonical identity is a Resource.

Conceptually:

```text
Resource =
    Kind
    Identity
    Data
    Relationships
```

A Resource participates in:

```text
ownership
references
graph relations
constraints
impact
packages
projections
```

This is likely the most fundamental concept.

---

# 14. Candidate Core primitive: Kind

A Kind is the nominal type of a Resource.

Examples:

```text
Module
UseCase
Rule
Agent
Threat
Release
```

The Core only needs to understand that a Kind exists and has a schema/semantic contract.

A Domain Extension defines:

```text
valid fields
valid relations
required fields
constraints
selectors/scopes
possibly projection metadata
```

---

# 15. Candidate Core primitive: Identity

Identity should be explicit and stable.

Conceptually:

```text
namespace / kind / name
```

Properties:

```text
- independent of file path
- globally unambiguous within origin/package scope
- deterministic
- usable in references
- preserved across projections
```

File location is storage, not identity.

---

# 16. Candidate Core primitive: Reference

Markitect should avoid untyped string references wherever possible.

Programming-language inspiration:

```text
Ref<Module>
Ref<Interface>
Ref<Rule>
```

Example:

```text
Module.dependencies
```

should only accept references to allowed Kinds.

References should be checked during compilation.

This creates a type system for the semantic graph.

---

# 17. Candidate Core primitive: Relation

Relations should be a central semantic concept.

Possible relation verbs:

```text
owns
dependsOn
implements
requires
produces
governs
exposes
consumes
```

A Domain Extension defines valid combinations.

Example:

```text
Module
  owns        UseCase
  owns        Aggregate
  exposes     Interface
  dependsOn   Module
```

The graph then becomes a semantic structure rather than merely a collection of generic links.

The Core can perform:

```text
reference resolution
cycle detection
impact propagation
scope validation
graph queries
```

without understanding the domain-specific business meaning.

---

# 18. References vs labels

A useful language rule:

> **References express semantics. Labels express classification.**

For example:

```yaml
metadata:
  labels:
    axis: intake
    criticality: high
    public: "true"
```

Labels are useful for:

```text
selection
grouping
querying
policy scope
```

They should not replace:

```text
ownership
dependency
contracts
semantic relationships
```

Those belong in explicit relations.

---

# 19. Candidate Core primitive: Constraint / Invariant

Constraints are analogous to:

```text
type rules
static assertions
refinement constraints
```

They express deterministic properties that must hold.

Examples:

```text
Module requires exactly one Intent.
```

```text
Module.dependencies may target:
- Core
- Modules with the same Axis
```

```text
Every public Interface must have exactly one owner.
```

Constraints make engineering policy compiler-checkable.

The design should investigate whether constraints are:

```text
declarative expressions
registered validators
plugin-defined predicates
or a combination
```

The Core should avoid embedding domain-specific condition logic directly.

---

# 20. Candidate Core primitive: Scope / Selector

Rules often apply to groups of resources.

Kubernetes-style selectors may be useful.

Example:

```yaml
scope:
  kind: Module
  matchLabels:
    regulated: "true"
```

Possible uses:

```text
apply policy to all matching Modules
query matching resources
select reconciliation scope
```

Important rule:

> **Selectors define scope, not semantic dependency.**

Selecting a Resource must not silently create graph relations.

---

# 21. Candidate Core primitive: Package

Packages distribute reusable canonical knowledge.

Examples:

```text
company-engineering
dotnet-ddd
release-policy
security-baseline
```

Packages should preserve canonical meaning and explicit identity.

A Package should not introduce hidden inheritance or implicit activation.

The existing Markitect package work already points in this direction.

---

# 22. Candidate Core primitive: Contract

`Contract` should be examined carefully.

It resembles concepts such as:

```text
interface
protocol
trait
type class
```

Example:

```text
ReleaseExecutor
```

may define:

```text
input:
  ReleaseCandidate

output:
  PublishedRelease
```

Different implementations may satisfy it.

This could be one of the strongest universal concepts in Markitect.

However, the implementer should determine whether `Contract` is truly Kernel-level or belongs to one domain.

---

# 23. Prefer composition over inheritance

Avoid deep inheritance trees such as:

```text
BaseEngineeringResource
    ↓
ProcessResource
    ↓
ReleaseProcessResource
    ↓
DotNetReleaseProcessResource
```

Prefer:

```text
composition
contracts
constraints
relations
```

The conceptual inspiration should be closer to:

```text
Go interfaces
Rust traits
protocols
composition-oriented systems
```

than traditional deep OOP inheritance.

---

# 24. Potential concept: Effects

Programming-language effect systems may provide useful inspiration for Processes.

A process or operation may declare:

```text
requires
produces
changes
externalEffects
```

Example:

```text
CreateModule

requires:
  ModuleDefinition

effects:
  Repository
  Documentation
  AzureDevOps

produces:
  Module
```

This could allow Markitect to understand structurally:

```text
what does the operation require?
what does it produce?
which external worlds may it change?
```

This is exploratory and should not be added until there is a concrete use case.

---

# 25. Potential concept: State Transition

Many engineering processes are naturally state transitions.

Examples:

```text
ReleaseCandidate
    ↓ Publish
PublishedRelease
```

```text
ModuleDefinition
    ↓ CreateModule
ReconciledModule
```

```text
CurrentEngineeringState
    ↓ ChangePlan
DesiredEngineeringState
```

It may be worth investigating whether a generic:

```text
Transition
```

is more fundamental than `Workflow`.

A Workflow could then be a composition of Transitions.

This should be evaluated rather than assumed.

---

# 26. Semantic IR

Plugins should ideally consume a stable semantic intermediate representation.

Conceptually:

```text
Authoring Syntax
       ↓
Compiler
       ↓
Canonical Semantic IR
       ↓
Domain Extensions / Adapters
```

The IR might contain normalized concepts such as:

```text
Resource
Identity
Kind
Relations
Resolved References
Constraints
Scope
Origin
Package
Labels
```

Domain-specific values remain available through typed schemas.

The IR should be independent of:

```text
YAML formatting
file ordering
comments
authoring syntax
```

---

# 27. Plugin compatibility as semantic ABI

A long-term goal could be:

```text
Plugin implements:
Markitect Semantic Model API v1
```

rather than:

```text
Plugin depends on:
Markitect YAML format v0.9
```

Then source syntax can evolve independently.

Potential future authoring forms:

```text
YAML
Visual editor
AI authoring
other serialization formats
```

while plugin contracts remain stable.

This is analogous to a compiler ABI/API boundary.

---

# 28. Domain Extensions

Domain Extensions define domain meaning.

Examples:

```text
Software Domain
    Module
    Product
    UseCase
    Aggregate
    Interface
    Invariant

Security Domain
    Threat
    Control
    TrustBoundary

Delivery Domain
    Release
    Deployment
    Environment
```

A Domain Extension may define:

```text
Kinds
Schemas
Valid Relations
Constraints
Selectors
Domain-specific diagnostics
```

It should not directly own external technology integration.

---

# 29. Adapter Plugins

Adapters map canonical meaning to technology-specific state.

Examples:

```text
Docs
Codex
Claude
Azure DevOps
GitHub
Jira
.NET
Java
Terraform
```

An Adapter consumes validated semantic IR.

It should never need to parse raw authoring YAML.

---

# 30. Proposed architecture layering

A useful structure to investigate:

```text
┌────────────────────────────────────────────┐
│             Authoring Languages            │
│                                            │
│ YAML       Visual UI         AI Author     │
└────────────────────┬───────────────────────┘
                     │
                     ▼
┌────────────────────────────────────────────┐
│             Markitect Compiler             │
│                                            │
│ Parse / Types / Refs / Constraints         │
└────────────────────┬───────────────────────┘
                     │
                     ▼
┌────────────────────────────────────────────┐
│          Canonical Semantic IR             │
│                                            │
│ Resources                                  │
│ Identity                                   │
│ Relations                                  │
│ Contracts                                  │
│ Constraints                                │
│ Scope                                      │
└────────────────────┬───────────────────────┘
                     │
          ┌──────────┴───────────┐
          ▼                      ▼
┌──────────────────┐   ┌────────────────────┐
│ Domain Extensions │   │     Adapters       │
│                   │   │                    │
│ Software          │   │ Docs               │
│ Security          │   │ Codex              │
│ Delivery          │   │ Claude             │
│ ...               │   │ Azure DevOps       │
└─────────┬─────────┘   │ .NET               │
          │             └─────────┬──────────┘
          └──────────────┬────────┘
                         ▼
                   Reconciliation
```

The exact boundary between Domain Extensions and IR/type registration needs careful design.

---

# 31. Reconciliation model

Once the canonical model is compiled, affected adapters can reconcile against it.

Conceptually:

```text
Canonical change
      ↓
Semantic impact
      ↓
Affected adapters
      ↓
Observe
      ↓
Plan
      ↓
Apply
      ↓
Verify
```

Plugins should not reconcile against each other's outputs.

They reconcile only against the semantic model.

---

# 32. Desired state vs observed state

The model should clearly distinguish:

```text
Desired State
    canonical model

Observed State
    adapter-specific external reality
```

An adapter may report:

```text
Expected:
Azure DevOps Area Modules/Intake/Survey

Observed:
missing
```

Markitect can then plan reconciliation.

Observed state must not silently mutate canonical desired state.

---

# 33. Important Kubernetes ideas to borrow

Potentially useful:

```text
versioned APIs
Kinds
metadata identity
spec vs status
extension model
selectors
controllers / reconciliation
desired vs observed state
```

These concepts fit Markitect well.

---

# 34. Kubernetes ideas to avoid copying blindly

Avoid:

```text
annotation sprawl
stringly typed references
labels replacing explicit semantics
very large YAML documents
hidden controller semantics
unnecessary eventual consistency
complex lifecycle machinery
Finalizer-like mechanisms without real need
```

The principle should be:

> **Borrow semantic ideas, not Kubernetes complexity.**

---

# 35. Programming-language ideas to borrow

Potentially useful:

```text
nominal types
typed references
interfaces / contracts
composition
static checking
constraints
packages
semantic IR
effect systems
state transitions
compiler phases
stable ABI/API boundaries
```

---

# 36. Programming-language ideas to avoid

Avoid:

```text
deep inheritance
implicit conversions
magic resolution rules
large hidden context
overloaded semantics
stringly typed relationships
```

The Markitect language should optimize for:

```text
clarity
local reasoning
explicitness
predictability
safe evolution
```

---

# 37. Candidate "Language Constitution"

A first draft of foundational rules:

1. **Everything with canonical identity is a Resource.**
2. **Every Resource has exactly one Kind and identity.**
3. **Kinds define valid structure and relations.**
4. **Relationships are explicit and typed.**
5. **References carry semantics; labels only classify.**
6. **Composition is preferred over inheritance.**
7. **Constraints express deterministic invariants.**
8. **Namespaces express ownership/scope, not inheritance.**
9. **Packages distribute canonical knowledge without changing its meaning.**
10. **Source syntax is not the semantic model.**
11. **Plugins consume validated semantic IR, not raw YAML.**
12. **Adapters never define canonical domain meaning.**
13. **Generated outputs are projections, never canonical sources.**
14. **Observed external state never silently changes desired canonical state.**
15. **Every change should have computable semantic impact wherever explicit relations make that possible.**

These should be treated as design hypotheses and reviewed critically.

---

# 38. Core question to answer

The most important design question is:

> **What are the minimal universal concepts Markitect Core must understand so that Module, Release, Agent, Threat, Service and future domains can all be modeled as vocabularies built on the same language?**

Current candidate primitives:

```text
Resource
Kind
Identity
Reference
Relation
Constraint
Scope
Package
```

Potential candidates requiring further evaluation:

```text
Contract
Transition
Effect
```

Everything else should ideally be expressed through Domain Extensions.

---

# 39. Questions for the implementer

Please review the current codebase and answer:

1. Which of these primitives already exist conceptually?
2. Which are currently mixed together accidentally?
3. Is the current `Resource` abstraction generic enough?
4. Is `Contract` truly Core-level?
5. Are `Rule`, `Workflow`, `Skill`, `Agent` actually Core concepts or one existing domain language?
6. How should Kinds be registered or defined safely?
7. How can typed references work without making the system overly complex?
8. Should Relations be first-class schema concepts?
9. How expressive should Constraints be?
10. Should Constraints be declarative, plugin-executed, or both?
11. Are selectors useful enough to justify Kernel support?
12. How should labels differ from semantic references?
13. What should the stable Semantic IR look like?
14. How should Domain Extensions interact with the IR?
15. What is the smallest plugin ABI that avoids raw-YAML coupling?
16. Can current graph/context/impact code be adapted naturally to this model?
17. Which existing concepts would need redesign?
18. What migration path would preserve current Markitect projects?
19. Which Kubernetes-inspired concepts are useful here?
20. Which Kubernetes concepts should explicitly be rejected?
21. Which programming-language concepts offer the most leverage?
22. Is `Transition` more fundamental than `Workflow`?
23. Would an Effect-like model improve process/reconciliation planning?
24. How can the Core remain small while still supporting powerful policy modeling?
25. What experiment would best validate the proposed kernel before broad implementation?

---

# 40. Requested review outcome

Please classify the proposed concepts as:

```text
Essential Kernel primitive
Useful Kernel primitive
Domain-level concept
Adapter-level concept
Interesting but premature
Unnecessary / harmful
```

Then propose a minimal **Markitect Language Kernel v1**.

The goal is to make the language:

```text
simple
orthogonal
powerful
predictable
extensible
easy to reason about
```

rather than feature-rich.

