# Markitect — AI-First Architecture as an Executable Engineering Constitution

## Status

Concept document for implementer review.

This document captures the primary personal/product use case discussed after the strategic positioning document.

The key idea is:

> **I do not want AI to invent the architecture every time it writes code. I want to design the architecture once, define how it may evolve, and let AI operate autonomously within those boundaries.**

This should be treated as a product-driving use case, not as a request to hard-code one architecture style into Markitect.

---

# 1. Working model

In an AI-first engineering workflow, humans increasingly stop operating primarily at code level.

The human/architect focuses on:

```text
boundaries
concepts
layering
invariants
extension points
allowed dependencies
change patterns
processes
exceptions
```

AI agents increasingly handle:

```text
implementation
refactoring
tests
mechanical changes
file creation
code movement
provider-specific updates
```

The intended operating model becomes:

```text
Human / Architect
        ↓
defines architecture + engineering rules
        ↓
Markitect
        ↓
compiled constraints + bounded context
        ↓
AI Agents
        ↓
implementation
        ↓
verification / reconciliation
```

---

# 2. Architecture should become "law", not advice

A typical architecture statement today is often prose such as:

> Use Cases should be implemented as Vertical Slices.

That is advisory unless something makes it canonical and enforceable.

The desired model is closer to:

```text
UseCase
- belongs to exactly one Module
- is Command or Query
- owns its Handler
- owns its slice-specific implementation
- may use explicitly allowed shared concepts
- must not create forbidden lateral dependencies
- follows required documentation/test policies
```

These should be canonical engineering statements.

From the same source Markitect can derive:

```text
human documentation
agent instructions
scaffolding hints
architecture checks
review context
impact
```

---

# 3. Strong statements are a feature

The engineering style intentionally prefers strong architectural statements:

```text
Core never depends on Modules.

A Use Case belongs to exactly one Module.

A Module exposes explicit Interfaces.

Cross-module access occurs only through defined contracts.

Domain Aggregates do not depend on Application concerns.

A Vertical Slice owns its Use-Case-specific implementation.
```

The value of such statements is that they reduce the valid design space.

Without them:

```text
many plausible implementations
```

With them:

```text
small set of architecturally valid implementations
```

This is particularly valuable for AI agents.

An agent is generally more useful when asked:

> Implement this task inside these architectural constraints.

than when asked:

> Decide again what architecture we should use.

---

# 4. Reduce AI degrees of freedom deliberately

Markitect should help define:

```text
what the agent may decide freely
what is already architecturally decided
what requires escalation
what is forbidden
```

Example task:

```text
Add ExportSurvey Use Case
```

The agent should ideally not need to rediscover:

```text
which layer?
which folder?
Command or Query?
where are tests?
which dependencies are legal?
where does documentation live?
which shared concepts are allowed?
```

The canonical engineering model should already answer these structural questions.

The agent then spends effort on the genuinely new domain behavior.

---

# 5. "Grammar of valid engineering changes"

A useful conceptual formulation:

> **Markitect defines the grammar of valid engineering changes.**

A programming language grammar limits which programs are syntactically/type-valid.

Similarly, Markitect should limit which engineering changes are structurally/architecturally valid.

Example:

```text
New UseCase
     ↓
belongs to Module
     ↓
Command | Query
     ↓
known Vertical Slice form
     ↓
required Handler
     ↓
required tests/docs
     ↓
allowed dependencies
```

The goal is to move failures from:

```text
runtime / later architectural review
```

toward:

```text
authoring / compile time
```

---

# 6. Four layers of engineering definition

A useful separation is:

## Architecture

What concepts exist and how may they relate?

```text
Module
Layer
UseCase
Aggregate
Interface
Product
```

## Policy

What rules apply?

```text
Module may depend only on Core.
Every Aggregate has an owner.
Every UseCase follows Vertical Slice.
```

## Process

How is the system changed correctly?

```text
AddModule
AddUseCase
Release
DeprecateInterface
MigrateModule
```

## Projection / Enforcement

How does the canonical model appear in concrete technologies?

```text
.NET
filesystem
documentation
Codex
Claude
Azure DevOps
CI
```

---

# 7. Architecture as executable engineering knowledge

Architecture should not mean only diagrams.

A stronger definition:

```text
Architecture
    =
Concepts
+ Relations
+ Constraints
+ Extension rules
+ Change processes
```

Then architecture becomes executable in the sense that it has concrete consequences:

```text
what is valid
what is invalid
where something belongs
what changes are affected
which checks must pass
which projections must reconcile
```

---

# 8. Code generation is not the main goal

Avoid reducing Markitect to scaffolding.

Bad end-state:

> Markitect emits fixed C# code templates for every pattern.

Preferred end-state:

```text
Markitect
    ↓
defines semantic structure + constraints

AI Agent
    ↓
chooses concrete implementation inside those constraints

Adapters/checks
    ↓
verify conformance
```

This preserves AI flexibility while keeping architecture stable.

---

# 9. Extension points should be explicit

A well-designed architecture often defines how future change is expected to happen.

Example:

```text
UseCase
├── required
│   └── Handler
│
└── extension points
    ├── Validator
    ├── Authorization
    ├── Mapping
    └── EventPublication
```

If validation is needed, the agent should use the known `Validator` extension point rather than inventing a new local mechanism.

This makes recurring change more mechanical and reduces architectural entropy.

---

# 10. Exceptions must be first-class

Strong rules without explicit exceptions encourage hidden violations.

A future model should support intentional exceptions:

```text
Architecture Exception
- violated rule
- subject
- rationale
- owner
- scope
- optional expiry/review date
```

Example:

```text
Rule:
Modules may depend only on Core.

Exception:
LegacyImport may depend on SharedKernel until Migration-X completes.
```

The deviation remains explicit and canonical rather than silently eroding architecture.

---

# 11. Normal change vs architecture change

Two classes of change should be distinguished.

## Normal engineering change

```text
follows existing architecture
```

Example:

```text
Add ExportSurvey Use Case
```

## Architecture change

```text
changes the engineering system itself
```

Example:

```text
Modules may now depend on SharedKernel.
```

An architecture change should trigger:

```text
impact
affected rules
affected docs
affected adapters
affected implementations
review
reconciliation
```

This distinction fits Markitect's existing change-management direction.

---

# 12. AI autonomy as a consequence of architectural clarity

Desired relationship:

```text
clearer canonical architecture
        ↓
fewer ambiguous structural decisions
        ↓
smaller relevant context
        ↓
higher safe agent autonomy
        ↓
longer unattended work
```

The goal is not to micromanage agents.

The goal is:

> **Constrain architecture strongly enough that agents can be trusted to operate freely inside it.**

---

# 13. Product statement for this use case

A focused definition:

> **Markitect lets architects define the shape of an engineering system strongly enough that AI agents can safely operate inside it autonomously.**

Expanded:

> **Markitect turns architectural intent, extension rules and engineering processes into a canonical executable model. AI agents receive the relevant part of that model for their task, implement changes within its constraints, and adapters verify that code, documentation and engineering systems remain reconciled with the intended architecture.**

---

# 14. Implementer review questions

1. Which current Markitect concepts already support this model?
2. Which architectural statements can already be represented as `Rule`, `Contract`, `Workflow`, relations or checks?
3. Which capabilities require a more general modeling kernel?
4. How should extension points be represented without over-modeling?
5. How should explicit architecture exceptions work?
6. How can normal changes and architecture changes be distinguished?
7. Can the current impact model propagate architecture-policy changes to affected resources/adapters?
8. Which conformance checks belong in Core and which belong in adapters?
9. How can Markitect avoid becoming a code-template engine?
10. What is the smallest real Konfyra-style experiment that proves this use case?

---

# 15. Core design principle

> **The purpose is not to make AI follow more prose. The purpose is to make the valid shape of future software explicit enough that AI does not need to reinvent it.**
