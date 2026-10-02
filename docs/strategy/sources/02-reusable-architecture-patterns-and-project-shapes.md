# Markitect — Reusable Architecture Patterns, Archetypes and Typed Project Shapes

## Status

Concept document for implementer review.

This document captures the idea that recurring architectural preferences should be reusable as **ongoing architectural contracts**, not merely copied as starter templates.

The motivating observation is that many projects repeatedly use the same styles:

```text
Application Layer rather than generic Services
Use Cases as Commands / Queries
one Handler per Use Case
Vertical Slices
Feature-as-Folder
Common for intentionally shared concepts
explicit Modules
DDD Aggregates
strong dependency rules
```

These patterns can potentially become reusable Markitect definitions.

---

# 1. Existing code already has an implicit grammar

Example observed structure:

```text
Application / UseCases
│
├── ChangeRequests
│   ├── ApplyChangeRequest
│   ├── CancelChangeRequest
│   ├── CommentOnChangeRequest
│   ├── GetChangeRequest
│   └── Common
│
├── Common
├── Dossiers
└── Specifications
```

This structure implies architectural rules such as:

```text
Application Logic is expressed as Use Cases.
Use Cases are grouped by Feature.
A Use Case is a Command or Query.
A Use Case owns a Handler.
Use-Case-specific code stays inside the Slice.
Shared concepts move to an explicit Common scope.
```

This is not merely folder naming.

It is an architecture pattern.

---

# 2. "Programming the space of valid software"

A useful concept:

> **Markitect does not program the implementation; it programs the space of architecturally valid implementations.**

Without such a model:

```text
Agent creates structure
        ↓
human later notices architectural drift
```

With typed architectural forms:

```text
Desired change
     ↓
architecture/type validation
     ↓
Agent implementation
     ↓
conformance verification
```

---

# 3. Distinguish Template, Pattern and Package

These concepts should not be conflated.

## Template

One-time bootstrap.

```text
Create Module
    ↓
initial files/directories
```

After creation, output becomes project-owned.

Question answered:

> How does it start?

## Pattern / Archetype

An ongoing semantic contract.

```text
Survey Module
conformsTo:
    KonfyraModule
```

If the Pattern changes, its instances may be affected.

Question answered:

> What shape is it allowed to have over time?

## Package

Versioned distribution of reusable knowledge.

```text
company-engineering-patterns@2.4
```

A package may contain:

```text
patterns
rules
contracts
processes
policies
```

Question answered:

> How is reusable engineering knowledge versioned and distributed?

---

# 4. Candidate example: VerticalSliceUseCase

A reusable pattern could conceptually express:

```text
VerticalSliceUseCase

parameters:
  name
  kind = Command | Query
  feature
  module

requires:
  exactly one Handler

rules:
  belongs to exactly one Feature
  may use feature-local Common
  may use module-level Common
  no lateral direct dependency on another UseCase
  tests belong to the Slice
```

A .NET adapter could map this to:

```text
Application/
  ChangeRequests/
    ApplyChangeRequest/
      ApplyChangeRequestCommand.cs
      ApplyChangeRequestHandler.cs
      ...
```

A documentation adapter could map it to human architecture docs.

Codex/Claude adapters could expose the operational rules.

The Pattern remains technology-neutral.

---

# 5. Semantic architecture, not folder architecture

Avoid making filesystem layout canonical.

Prefer:

```text
Semantic:
UseCase belongs to Feature.
UseCase owns Handler.
```

Then a .NET adapter may choose:

```text
ChangeRequests/
  ApplyChangeRequest/
```

A Java adapter may choose a package representation.

The canonical model should describe architecture meaning.

Adapters describe technology-specific layout.

---

# 6. Project architectures as composition

A preferred architecture could be a composition of reusable forms.

Conceptually:

```text
KonfyraModule
    =
ModuleBoundary
+ CleanArchitectureLayers
+ VerticalSliceApplication
+ DDDDomain
+ DocumentationPolicy
+ TestingPolicy
```

A concrete module then conforms to that composed architecture:

```text
Survey Module
    conformsTo KonfyraModule
```

This resembles:

```text
traits
interfaces
composition
```

more than classical inheritance.

---

# 7. Avoid deep architecture inheritance

Avoid:

```text
BaseModule
    ↓
DDDModule
    ↓
CleanDDDModule
    ↓
KonfyraCleanDDDModule
    ↓
SurveyModule
```

Prefer explicit composition:

```text
SurveyModule
  conformsTo:
    - ModuleBoundary
    - VerticalSliceApplication
    - DDDDomain
    - KonfyraDocumentationPolicy
```

This should make reuse more local and understandable.

---

# 8. Clean Architecture as a model

Instead of documenting Clean Architecture as prose only:

```text
Layer Domain
Layer Application
Layer Infrastructure
Layer API
```

with explicit relations:

```text
Domain
    dependsOn nothing outward

Application
    may depend on Domain

Infrastructure
    implements required contracts

API
    invokes Application
```

Additional preferred style:

```text
Application
    consistsOf UseCases

UseCase
    follows VerticalSlice
```

This makes the architecture partially compiler-checkable.

---

# 9. Typed architectural forms

Patterns may act like semantic types.

Conceptually:

```text
ApplyChangeRequest : CommandUseCase
GetChangeRequest   : QueryUseCase
```

Then rules can differ by type:

```text
CommandUseCase
    may mutate state

QueryUseCase
    must not mutate state
```

This is not necessarily proposed syntax.

The important idea is that recurring engineering forms can have explicit type-level semantics.

---

# 10. Extension points as part of a Pattern

Patterns should not only forbid invalid forms.

They should communicate how valid extension is expected to happen.

Example:

```text
VerticalSliceUseCase
├── required
│   └── Handler
│
└── optional extension points
    ├── Validator
    ├── Authorization
    ├── Mapping
    └── EventPublication
```

This provides agents with approved mechanisms instead of forcing them to invent new structures.

---

# 11. Architecture packages

Reusable personal/company styles could become packages.

Examples:

```text
glacius/software-architecture
konfyra-engineering
company/release-policy
company/modular-ddd
```

Possible contents:

```text
DDDModule
CleanArchitectureLayers
VerticalSliceUseCase
AggregatePattern
EntityPattern
InterfacePolicy
DocumentationPolicy
ReleasePolicy
TestingPolicy
```

This enables:

> **Architecture reuse as code.**

Not by copying a starter repository, but by reusing architectural rules.

---

# 12. Pattern versioning

Example:

```text
VerticalSliceUseCase v1
```

requires:

```text
Command
Handler
Tests
```

Later:

```text
VerticalSliceUseCase v2
```

also requires:

```text
Validator
```

Markitect could determine:

```text
Pattern changed
      ↓
affected instances
      ↓
migration / reconciliation plan
```

This resembles:

> **Schema migration for architecture.**

This is potentially powerful and should be explored carefully.

---

# 13. Template vs ongoing conformance

A classic project template only produces:

```text
initial state
```

Afterward the project may drift.

Markitect Patterns could provide:

```text
initial state
        +
ongoing architectural contract
```

Concise distinction:

```text
Template:
How does it start?

Pattern:
What is it allowed to become?
```

---

# 14. Possible relation to Domain Extensions

Open design question:

Should `Pattern` itself be a Core concept?

Possible alternatives:

```text
A. Core Pattern primitive

B. Domain-defined Kind using existing Core primitives

C. Package convention composed from Relations + Constraints + Contracts
```

Do not introduce `Kind: Pattern` merely because the concept is useful in discussion.

The implementer should first determine whether existing primitives can express the same semantics cleanly.

---

# 15. Conformance model

A useful capability would be:

```text
Instance
  conformsTo Pattern
```

Markitect could then validate:

```text
required relations
required children
constraints
allowed extension points
forbidden relationships
```

Adapters may additionally verify technology-specific conformance.

---

# 16. Example personal architecture style

A reusable style may encode:

```text
Application Layer rather than generic Service dumping grounds

Application Logic as explicit Use Cases

Commands and Queries as distinct UseCase forms

one Handler per UseCase

Vertical Slice ownership

Common only for genuinely shared concepts

explicit Module boundaries

stable Core abstractions

strong invariants

composition over inheritance

repeating changes should have known extension points
```

The goal is not to encode a person's taste as vague prompt prose.

The goal is to make recurring architectural decisions explicit and reusable.

---

# 17. Relation to AI-agent autonomy

A reusable Pattern gives an AI agent a known expansion path.

Instead of:

> "Create a reasonable new feature."

the agent receives:

```text
Create:
  ExportSurvey : QueryUseCase

inside:
  Survey : KonfyraModule

conforming to:
  VerticalSliceUseCase
  TestingPolicy
  DocumentationPolicy
```

This significantly reduces architectural invention during implementation.

---

# 18. Implementer review questions

1. Does the current model already have enough primitives for ongoing Pattern conformance?
2. Is a new `Pattern` primitive necessary?
3. How should Pattern composition work?
4. How should conflicts between composed Patterns be reported?
5. How should Pattern parameters work, if at all?
6. How should versioned Pattern changes propagate to instances?
7. Can existing Package semantics distribute Patterns naturally?
8. How should Templates differ technically from ongoing Patterns?
9. How should adapter-specific layouts remain outside canonical semantics?
10. Can extension points be represented cleanly without creating a large meta-language?
11. What is the smallest experiment using the shown Vertical Slice structure?
12. What would make this too complicated or too similar to a full architecture DSL?

---

# 19. Core principle

> **A reusable architecture should not only generate a project's starting state; it should define the valid shape of its future evolution.**
